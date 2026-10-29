"use client";

import Image from "next/image";
import Link from "next/link";
import { useEffect, useRef, useState } from "react";
import { BrandLogo } from "@/components/brand-logo";

export default function HomePage() {
  const pageRef = useRef<HTMLElement>(null);
  const [isWelcomeVisible, setIsWelcomeVisible] = useState(true);
  const [isWelcomeExiting, setIsWelcomeExiting] = useState(false);

  useEffect(() => {
    if (window.matchMedia("(prefers-reduced-motion: reduce)").matches) {
      const skipTimer = window.setTimeout(() => setIsWelcomeVisible(false), 0);
      return () => window.clearTimeout(skipTimer);
    }

    const exitTimer = window.setTimeout(() => setIsWelcomeExiting(true), 2350);
    const finishTimer = window.setTimeout(() => setIsWelcomeVisible(false), 2950);

    return () => {
      window.clearTimeout(exitTimer);
      window.clearTimeout(finishTimer);
    };
  }, []);

  useEffect(() => {
    const page = pageRef.current;
    const story = page?.querySelector<HTMLElement>("#story");
    const stage = page?.querySelector<HTMLElement>(".cinematic-stage");
    const progressBar = page?.querySelector<HTMLElement>("#progress");
    const scenes = Array.from(page?.querySelectorAll<HTMLElement>(".story-scene") ?? []);
    const parallaxImages = Array.from(
      page?.querySelectorAll<HTMLElement>(".parallax-image[data-speed]") ?? [],
    );
    const depthTargets = Array.from(page?.querySelectorAll<HTMLElement>("[data-depth]") ?? []);

    if (!page || !story || !stage || !progressBar || scenes.length === 0) {
      throw new Error("The Ayuprama landing story requires its stage, scenes, and progress element.");
    }

    const pageElement = page;
    const storyElement = story;
    const stageElement = stage;
    const progressElement = progressBar;
    const reducedMotion = window.matchMedia("(prefers-reduced-motion: reduce)");
    const sceneMotion = new Map(
      parallaxImages.map((element) => [element, { current: 0, target: 0 }]),
    );
    let targetProgress = 0;
    let cameraProgress = 0;
    let lastDominantIndex = -1;
    let animationFrame = 0;
    let scrollUpdateQueued = false;
    let pointerFrame = 0;
    let pointerX = 0;
    let pointerY = 0;

    page.classList.add("js-ready");
    story.style.setProperty("--scene-count", String(scenes.length));

    function clamp(value: number, min: number, max: number) {
      return Math.min(max, Math.max(min, value));
    }

    function smoothstep(edge0: number, edge1: number, value: number) {
      const amount = clamp((value - edge0) / (edge1 - edge0), 0, 1);
      return amount * amount * (3 - 2 * amount);
    }

    function revealScene(scene: HTMLElement) {
      scene.querySelectorAll<HTMLElement>(".reveal").forEach((element) => {
        element.classList.add("is-visible");
      });
    }

    function updateScenePositions(value: number) {
      if (reducedMotion.matches) {
        scenes.forEach((scene) => {
          scene.classList.add("scene-active");
          scene.inert = false;
          scene.removeAttribute("aria-hidden");
          revealScene(scene);
        });
        return;
      }

      const lastIndex = scenes.length - 1;
      const boundedProgress = clamp(value, 0, lastIndex);
      const currentIndex = Math.min(Math.floor(boundedProgress), lastIndex);
      const hasNext = currentIndex < lastIndex;
      const fraction = hasNext ? boundedProgress - currentIndex : 0;
      const transition = hasNext ? smoothstep(0.3, 0.7, fraction) : 0;
      const dominantIndex = hasNext && transition > 0.5 ? currentIndex + 1 : currentIndex;

      scenes.forEach((scene, index) => {
        let opacity = 0;
        let depth = -760;
        let vertical = 0;
        let rotateX = 0;
        let rotateY = 0;
        let scale = 0.78;

        if (index === currentIndex) {
          opacity = 1 - transition;
          depth = -transition * 300;
          vertical = -transition * 64;
          rotateX = -transition * 5;
          rotateY = transition * (index % 2 === 0 ? -8 : 8);
          scale = 1 + transition * 0.08;
        } else if (hasNext && index === currentIndex + 1) {
          const entrance = 1 - transition;
          opacity = transition;
          depth = -entrance * 560;
          vertical = entrance * 94;
          rotateX = entrance * 8;
          rotateY = entrance * (index % 2 === 0 ? 9 : -9);
          scale = 1 - entrance * 0.14;
        }

        scene.style.setProperty("--scene-opacity", String(opacity));
        scene.style.setProperty("--scene-z", `${depth}px`);
        scene.style.setProperty("--scene-y", `${vertical}px`);
        scene.style.setProperty("--scene-rotate-x", `${rotateX}deg`);
        scene.style.setProperty("--scene-rotate-y", `${rotateY}deg`);
        scene.style.setProperty("--scene-scale", String(scale));

        const active = index === dominantIndex;
        const wasActive = scene.classList.contains("scene-active");
        if (wasActive !== active || scene.getAttribute("aria-hidden") === null) {
          scene.classList.toggle("scene-active", active);
          scene.inert = !active;
          scene.setAttribute("aria-hidden", String(!active));
          if (active) revealScene(scene);
        }
      });

      if (dominantIndex !== lastDominantIndex) {
        lastDominantIndex = dominantIndex;
        stageElement.setAttribute("aria-label", `Bagian ${dominantIndex + 1} dari ${scenes.length}`);
      }
    }

    function animateCamera() {
      if (reducedMotion.matches) {
        cameraProgress = targetProgress;
      } else {
        cameraProgress += (targetProgress - cameraProgress) * 0.12;
        if (Math.abs(targetProgress - cameraProgress) < 0.001) cameraProgress = targetProgress;
      }

      updateScenePositions(cameraProgress);

      let parallaxSettling = false;
      for (const [image, motion] of sceneMotion) {
        motion.current += (motion.target - motion.current) * (reducedMotion.matches ? 1 : 0.12);
        if (Math.abs(motion.target - motion.current) > 0.1) parallaxSettling = true;
        image.style.setProperty("--parallax-offset", `${motion.current}px`);
      }

      if (Math.abs(targetProgress - cameraProgress) > 0.001 || parallaxSettling) {
        animationFrame = window.requestAnimationFrame(animateCamera);
      } else {
        animationFrame = 0;
      }
    }

    function updateScrollTarget() {
      const storyTop = storyElement.getBoundingClientRect().top + window.scrollY;
      const sceneHeight = Math.max(1, window.innerHeight);
      targetProgress = clamp((window.scrollY - storyTop) / sceneHeight, 0, scenes.length - 1);

      const documentHeight = document.documentElement.scrollHeight - sceneHeight;
      const pageProgress = documentHeight > 0
        ? clamp(window.scrollY / documentHeight, 0, 1)
        : 0;
      progressElement.style.width = `${pageProgress * 100}%`;
      progressElement.setAttribute("aria-valuenow", String(Math.round(pageProgress * 100)));

      for (const image of parallaxImages) {
        const speed = Number(image.dataset.speed);
        const scene = image.closest<HTMLElement>(".story-scene");
        const sceneIndex = scene ? scenes.indexOf(scene) : -1;
        const motion = sceneMotion.get(image);
        if (Number.isFinite(speed) && sceneIndex >= 0 && motion) {
          motion.target = reducedMotion.matches
            ? 0
            : clamp((sceneIndex - targetProgress) * sceneHeight * speed * 0.42, -180, 180);
        }
      }

      if (reducedMotion.matches) cameraProgress = targetProgress;
      if (!animationFrame) animationFrame = window.requestAnimationFrame(animateCamera);
    }

    function queueScrollUpdate() {
      if (scrollUpdateQueued) return;
      scrollUpdateQueued = true;
      window.requestAnimationFrame(() => {
        updateScrollTarget();
        scrollUpdateQueued = false;
      });
    }

    function navigateToScene(scene: HTMLElement) {
      const index = scenes.indexOf(scene);
      if (index < 0) return;

      const storyTop = storyElement.getBoundingClientRect().top + window.scrollY;
      const top = reducedMotion.matches
        ? scene.getBoundingClientRect().top + window.scrollY
        : storyTop + index * window.innerHeight;
      const hash = `#${scene.id}`;
      if (window.location.hash !== hash) window.history.pushState(null, "", hash);

      if (reducedMotion.matches) {
        window.scrollTo(0, top);
      } else {
        window.scrollTo({ top, behavior: "smooth" });
      }
    }

    function handleClick(event: MouseEvent) {
      if (
        event.defaultPrevented
        || event.button !== 0
        || event.metaKey
        || event.ctrlKey
        || event.shiftKey
        || event.altKey
        || !(event.target instanceof Element)
      ) return;

      const link = event.target.closest<HTMLAnchorElement>('a[href^="#"]');
      if (!link || !pageElement.contains(link)) return;

      const id = link.getAttribute("href")?.slice(1);
      const scene = scenes.find((candidate) => candidate.id === id);
      if (!scene) return;

      event.preventDefault();
      navigateToScene(scene);
    }

    function handlePointerMove(event: PointerEvent) {
      if (reducedMotion.matches) return;

      pointerX = (event.clientX / window.innerWidth - 0.5) * 2;
      pointerY = (event.clientY / window.innerHeight - 0.5) * 2;
      if (pointerFrame) return;

      pointerFrame = window.requestAnimationFrame(() => {
        for (const target of depthTargets) {
          const depth = Number(target.dataset.depth);
          if (!Number.isFinite(depth)) continue;
          target.style.setProperty("--depth-x", `${pointerX * depth * 220}px`);
          target.style.setProperty("--depth-y", `${pointerY * depth * 220}px`);
        }
        pointerFrame = 0;
      });
    }

    function handleMotionPreferenceChange() {
      lastDominantIndex = -1;
      if (animationFrame) window.cancelAnimationFrame(animationFrame);
      animationFrame = 0;
      updateScrollTarget();
    }

    pageElement.addEventListener("click", handleClick);
    window.addEventListener("scroll", queueScrollUpdate, { passive: true });
    window.addEventListener("resize", queueScrollUpdate);
    window.addEventListener("pointermove", handlePointerMove, { passive: true });
    reducedMotion.addEventListener("change", handleMotionPreferenceChange);

    if (window.location.hash) {
      const initialScene = scenes.find((scene) => `#${scene.id}` === window.location.hash);
      if (initialScene) {
        const initialIndex = scenes.indexOf(initialScene);
        const initialTop = reducedMotion.matches
          ? initialScene.getBoundingClientRect().top + window.scrollY
          : story.offsetTop + initialIndex * window.innerHeight;
        window.scrollTo(0, initialTop);
        targetProgress = initialIndex;
        cameraProgress = initialIndex;
      }
    }

    updateScrollTarget();

    pageElement.querySelectorAll<HTMLImageElement>("img").forEach((image) => {
      image.addEventListener("error", () => {
        console.error(`Landing page image failed to load: ${image.getAttribute("src")}`);
      }, { once: true });
    });

    return () => {
      if (animationFrame) window.cancelAnimationFrame(animationFrame);
      if (pointerFrame) window.cancelAnimationFrame(pointerFrame);
      pageElement.removeEventListener("click", handleClick);
      window.removeEventListener("scroll", queueScrollUpdate);
      window.removeEventListener("resize", queueScrollUpdate);
      window.removeEventListener("pointermove", handlePointerMove);
      reducedMotion.removeEventListener("change", handleMotionPreferenceChange);
      page.classList.remove("js-ready");
      storyElement.style.removeProperty("--scene-count");
      scenes.forEach((scene) => {
        scene.classList.remove("scene-active");
        scene.inert = false;
        scene.removeAttribute("aria-hidden");
        scene.removeAttribute("style");
      });
      parallaxImages.forEach((image) => image.style.removeProperty("--parallax-offset"));
    };
  }, []);

  return (
    <main ref={pageRef} className="page">
      {isWelcomeVisible && (
        <div className={`welcome-screen${isWelcomeExiting ? " is-exiting" : ""}`} role="status" aria-label="Selamat datang ke laboratorium umat manusia" aria-live="polite" aria-atomic="true">
          <BrandLogo alt="" width={84} height={70} priority className="welcome-logo" />
          <p className="welcome-title" aria-hidden="true">
            <span>Selamat</span>{" "}
            <span>datang</span>{" "}
            <span>ke</span>
            <br />
            <span>laboratorium</span>{" "}
            <span className="welcome-accent">umat</span>{" "}
            <span className="welcome-accent">manusia.</span>
          </p>
          <div className="welcome-progress" aria-hidden="true">
            <span />
          </div>
        </div>
      )}

      <div className="landing-content" aria-hidden={isWelcomeVisible} inert={isWelcomeVisible}>
      <a className="skip-link" href="#home">Langsung ke konten</a>
      <div id="progress" role="progressbar" aria-label="Progres halaman" aria-valuemin={0} aria-valuemax={100} aria-valuenow={0} />

      <div className="cinematic-story" id="story">
        <div className="cinematic-stage" role="region" aria-label="Cerita Ayuprama">
          <div className="spatial-floor" aria-hidden="true" />
          <section className="hero scene story-scene" id="home" aria-labelledby="hero-title">
            <div className="hero-image parallax-image" data-speed=".16" aria-hidden="true" />
            <div className="hero-vignette" aria-hidden="true" />
            <div className="orb orb-one" data-depth=".035" aria-hidden="true" />
            <div className="orb orb-two" data-depth="-.025" aria-hidden="true" />
            <div className="hero-content reveal">
              <p className="eyebrow"><span className="eyebrow-line" /> PRIOR AUTHORIZATION & KELENGKAPAN KLAIM</p>
              <h1 id="hero-title">Permohonan lebih terarah.<br /><em>Klaim lebih lengkap.</em></h1>
              <p className="hero-copy">
                Satukan informasi klinis, persyaratan prior authorization, dan
                pemeriksaan kelengkapan klaim dalam alur kerja yang jelas.
              </p>
              <div className="hero-actions">
                <a className="button button-light" href="#workflow">Lihat alur kerja <span aria-hidden="true">→</span></a>
                <a className="text-link" href="#solution"><span className="play" aria-hidden="true">↓</span> Kenali Ayuprama</a>
              </div>
            </div>
            <div className="floating-card" data-depth=".06" aria-label="Contoh alur, bukan data pasien">
              <div className="card-top"><span className="tiny-icon" aria-hidden="true">✚</span><span className="status">CONTOH ALUR</span></div>
              <div className="patient-row"><div className="avatar" aria-hidden="true">PA</div><div><strong>Permohonan</strong><small>Ilustrasi, bukan data pasien</small></div></div>
              <div className="approval-state"><span className="check" aria-hidden="true">✓</span><div><strong>Siap ditinjau</strong><small>Perlu verifikasi reviewer</small></div></div>
              <div className="mini-progress"><span /></div>
              <div className="card-foot"><span>Kelengkapan dokumen</span><strong>Checklist</strong></div>
            </div>
            <div className="hero-bottom" aria-hidden="true">
              <span>01 / ALUR KERJA YANG LEBIH TERHUBUNG</span>
              <span className="scroll-cue">GULIR UNTUK MENJELAJAHI <b>↓</b></span>
            </div>
          </section>

          <section className="intro section-pad story-scene" id="solution" aria-labelledby="solution-title">
            <div className="intro-copy reveal">
              <p className="eyebrow dark"><span className="eyebrow-line" /> SATU ALUR YANG TERHUBUNG</p>
              <h2 id="solution-title">Dari kebutuhan klinis<br />ke <em>berkas siap ditinjau.</em></h2>
              <p className="body-copy">
                Ayuprama membantu tim menghubungkan informasi klinis, langkah
                prior authorization, dan pemeriksaan kelengkapan klaim dalam
                satu proses terpandu.
              </p>
              <a className="inline-link" href="#workflow">Lihat tahapan alur <span aria-hidden="true">↗</span></a>
            </div>
            <div className="ecosystem-stage reveal" aria-label="Keterhubungan informasi permohonan, persyaratan, dan pemeriksaan klaim">
              <div className="ecosystem-image parallax-image" data-speed=".08" aria-hidden="true" />
              <div className="stage-glow" aria-hidden="true" />
              <div className="data-node node-left" data-depth=".035"><span className="node-symbol" aria-hidden="true">⌘</span><strong>Informasi klinis</strong><small>Catatan · Diagnosis · Layanan</small></div>
              <div className="data-node node-top" data-depth="-.025"><span className="node-symbol" aria-hidden="true">◇</span><strong>Persyaratan payer</strong><small>Manfaat · Kebijakan · Kriteria</small></div>
              <div className="data-node node-right" data-depth=".045"><span className="node-symbol" aria-hidden="true">✓</span><strong>Pemeriksaan klaim</strong><small>Kode · Dokumen · Kelayakan</small></div>
              <div className="core-ring" aria-hidden="true"><BrandLogo alt="" width={52} height={44} className="core-logo" /><span>AYUPRAMA</span></div>
              <svg className="connectors" viewBox="0 0 700 460" preserveAspectRatio="none" aria-hidden="true">
                <path d="M110 180 C180 180 210 245 300 250" />
                <path d="M350 100 C350 160 350 185 350 220" />
                <path d="M590 190 C520 190 485 245 400 250" />
              </svg>
            </div>
          </section>

          <section className="workflow scene story-scene" id="workflow" aria-labelledby="workflow-title">
            <div className="workflow-bg parallax-image" data-speed=".07" aria-hidden="true" />
            <div className="workflow-overlay" aria-hidden="true" />
            <div className="workflow-inner section-pad">
              <div className="workflow-heading reveal">
                <p className="eyebrow"><span className="eyebrow-line" /> LANGKAH YANG JELAS UNTUK SETIAP PERMOHONAN</p>
                <h2 id="workflow-title">Kurangi pencarian.<br /><em>Tambah kejelasan.</em></h2>
                <p>Susun proses administrasi menjadi tahapan yang dapat dipahami, diperiksa, dan ditindaklanjuti.</p>
              </div>
              <div className="steps">
                <article className="step reveal"><span className="step-number">01</span><div className="step-icon" aria-hidden="true">▤</div><h3>Siapkan konteks</h3><p>Satukan catatan klinis dan dokumen pendukung untuk permohonan.</p><span className="step-line" /></article>
                <article className="step reveal"><span className="step-number">02</span><div className="step-icon" aria-hidden="true">⌕</div><h3>Periksa persyaratan</h3><p>Tinjau informasi dan kriteria yang diperlukan sebelum pengajuan.</p><span className="step-line" /></article>
                <article className="step reveal"><span className="step-number">03</span><div className="step-icon" aria-hidden="true">✓</div><h3>Tinjau kelengkapan</h3><p>Kenali item yang belum tersedia dan tentukan langkah berikutnya.</p><span className="step-line" /></article>
              </div>
            </div>
            <div className="workflow-foot"><span>KEPUTUSAN TETAP PADA PENINJAU</span><span>KONTEKS KLINIS + KEJELASAN ADMINISTRATIF</span></div>
          </section>

          <section className="benefits section-pad story-scene" id="benefits" aria-labelledby="benefits-title">
            <div className="benefit-visual reveal">
              <Image
                src="/images/ayuprama-lab-detail.jpg"
                alt="Peralatan laboratorium medis"
                width={960}
                height={640}
                sizes="(max-width: 700px) 86vw, 48vw"
              />
              <div className="visual-caption"><span className="pulse" aria-hidden="true" /> Informasi ditinjau oleh manusia</div>
            </div>
            <div className="benefit-copy reveal">
              <p className="eyebrow dark"><span className="eyebrow-line" /> UNTUK PROSES TINJAUAN</p>
              <h2 id="benefits-title">Dibangun untuk alur<br /><em>administratif yang jelas.</em></h2>
              <p className="body-copy">
                Tim layanan kesehatan dapat melihat kekurangan dokumen lebih
                awal, memahami kebutuhan permohonan, dan menjaga keputusan
                tetap terlihat oleh pihak yang bertanggung jawab.
              </p>
              <div className="benefit-list">
                <div><span className="list-icon" aria-hidden="true">✓</span><p><strong>Kenali kekurangan lebih awal</strong><small>Periksa dokumentasi yang belum lengkap sebelum pengajuan.</small></p></div>
                <div><span className="list-icon" aria-hidden="true">⌁</span><p><strong>Pahami langkah berikutnya</strong><small>Lihat status permohonan dan item yang perlu ditindaklanjuti.</small></p></div>
                <div><span className="list-icon" aria-hidden="true">✓</span><p><strong>Pertahankan tinjauan manusia</strong><small>Dukung proses review tanpa menggantikan keputusan klinis atau payer.</small></p></div>
              </div>
            </div>
          </section>

          <section className="cta-section scene story-scene" id="contact" aria-labelledby="contact-title">
            <div className="cta-image parallax-image" data-speed=".1" aria-hidden="true" />
            <div className="cta-overlay" aria-hidden="true" />
            <div className="cta-content reveal">
              <p className="eyebrow"><span className="eyebrow-line" /> RUANG KERJA AYUPRAMA</p>
              <h2 id="contact-title">Jadikan langkah berikutnya<br /><em>lebih jelas.</em></h2>
              <p>Masuk untuk mengelola permohonan prior authorization dan meninjau kelengkapan klaim.</p>
              <Link href="/login" className="button button-light">Masuk ke Ayuprama <span aria-hidden="true">→</span></Link>
              <small className="disclaimer">Ayuprama mendukung pemeriksaan administrasi, bukan keputusan klinis atau payer.</small>
            </div>
            <div className="cta-orbit orbit-a" aria-hidden="true" />
            <div className="cta-orbit orbit-b" aria-hidden="true" />
          </section>
        </div>
      </div>

      <footer className="site-footer">
        <Link className="brand" href="#home" aria-label="Ayuprama, kembali ke beranda">
          <BrandLogo alt="" width={38} height={32} className="brand-mark" />
          <span>Ayuprama</span>
        </Link>
        <span>Ruang kerja prior authorization dan kelengkapan klaim</span>
        <a href="#home">Kembali ke atas ↑</a>
      </footer>
      </div>
    </main>
  );
}
