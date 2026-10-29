const story = document.querySelector('#story');
const stage = document.querySelector('.cinematic-stage');
const progressBar = document.querySelector('#progress');
const scenes = [...document.querySelectorAll('.story-scene')];
const parallaxImages = [...document.querySelectorAll('.parallax-image[data-speed]')];
const depthTargets = [...document.querySelectorAll('[data-depth]')];
const reducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)');

if (!(story instanceof HTMLElement) || !(stage instanceof HTMLElement) || !(progressBar instanceof HTMLElement) || scenes.length === 0) {
  throw new Error('The cinematic story requires its story, stage, scenes, and progress elements.');
}

document.documentElement.classList.add('js-ready');
story.style.setProperty('--scene-count', String(scenes.length));

const sceneMotion = new Map(
  parallaxImages.map((element) => [element, { current: 0, target: 0 }]),
);
let targetProgress = 0;
let cameraProgress = 0;
let lastDominantIndex = -1;
let animationFrame = 0;
let scrollUpdateQueued = false;
let pointerX = 0;
let pointerY = 0;

function clamp(value, min, max) {
  return Math.min(max, Math.max(min, value));
}

function smoothstep(edge0, edge1, value) {
  const amount = clamp((value - edge0) / (edge1 - edge0), 0, 1);
  return amount * amount * (3 - 2 * amount);
}

function revealScene(scene) {
  scene.querySelectorAll('.reveal').forEach((element) => element.classList.add('is-visible'));
}

function updateScenePositions(value) {
  const lastIndex = scenes.length - 1;
  const boundedProgress = clamp(value, 0, lastIndex);
  const currentIndex = Math.min(Math.floor(boundedProgress), lastIndex);
  const hasNext = currentIndex < lastIndex;
  const fraction = hasNext ? boundedProgress - currentIndex : 0;
  const transition = hasNext ? smoothstep(.3, .7, fraction) : 0;
  const dominantIndex = hasNext && transition > .5 ? currentIndex + 1 : currentIndex;

  scenes.forEach((scene, index) => {
    let opacity = 0;
    let depth = -760;
    let vertical = 0;
    let rotateX = 0;
    let rotateY = 0;
    let scale = .78;

    if (index === currentIndex) {
      opacity = 1 - transition;
      depth = -transition * 300;
      vertical = -transition * 64;
      rotateX = -transition * 5;
      rotateY = transition * (index % 2 === 0 ? -8 : 8);
      scale = 1 + transition * .08;
    } else if (hasNext && index === currentIndex + 1) {
      const entrance = 1 - transition;
      opacity = transition;
      depth = -entrance * 560;
      vertical = entrance * 94;
      rotateX = entrance * 8;
      rotateY = entrance * (index % 2 === 0 ? 9 : -9);
      scale = 1 - entrance * .14;
    }

    scene.style.setProperty('--scene-opacity', String(reducedMotion.matches ? 1 : opacity));
    scene.style.setProperty('--scene-z', `${reducedMotion.matches ? 0 : depth}px`);
    scene.style.setProperty('--scene-y', `${reducedMotion.matches ? 0 : vertical}px`);
    scene.style.setProperty('--scene-rotate-x', `${reducedMotion.matches ? 0 : rotateX}deg`);
    scene.style.setProperty('--scene-rotate-y', `${reducedMotion.matches ? 0 : rotateY}deg`);
    scene.style.setProperty('--scene-scale', String(reducedMotion.matches ? 1 : scale));

    const active = reducedMotion.matches || index === dominantIndex;
    const wasActive = scene.classList.contains('scene-active');
    if (wasActive !== active || scene.getAttribute('aria-hidden') === null) {
      scene.classList.toggle('scene-active', active);
      scene.inert = !active;
      scene.setAttribute('aria-hidden', String(!active));
      if (active) revealScene(scene);
    }
  });

  if (dominantIndex !== lastDominantIndex) {
    lastDominantIndex = dominantIndex;
    stage.setAttribute('aria-label', `Scene ${dominantIndex + 1} of ${scenes.length}`);
  }
}

function animateCamera() {
  if (reducedMotion.matches) {
    cameraProgress = targetProgress;
  } else {
    cameraProgress += (targetProgress - cameraProgress) * .12;
    if (Math.abs(targetProgress - cameraProgress) < .001) cameraProgress = targetProgress;
  }

  updateScenePositions(cameraProgress);

  let parallaxSettling = false;
  for (const [image, motion] of sceneMotion) {
    motion.current += (motion.target - motion.current) * (reducedMotion.matches ? 1 : .12);
    if (Math.abs(motion.target - motion.current) > .1) parallaxSettling = true;
    image.style.setProperty('--parallax-offset', `${motion.current}px`);
  }

  if (Math.abs(targetProgress - cameraProgress) > .001 || parallaxSettling) {
    animationFrame = window.requestAnimationFrame(animateCamera);
  } else {
    animationFrame = 0;
  }
}

function updateScrollTarget() {
  const storyTop = story.getBoundingClientRect().top + window.scrollY;
  const sceneHeight = Math.max(1, window.innerHeight);
  targetProgress = clamp((window.scrollY - storyTop) / sceneHeight, 0, scenes.length - 1);

  const documentHeight = document.documentElement.scrollHeight - sceneHeight;
  const pageProgress = documentHeight > 0 ? clamp(window.scrollY / documentHeight, 0, 1) : 0;
  progressBar.style.width = `${pageProgress * 100}%`;

  for (const image of parallaxImages) {
    const speed = Number(image.dataset.speed);
    const scene = image.closest('.story-scene');
    const sceneIndex = scenes.indexOf(scene);
    if (Number.isFinite(speed) && sceneIndex >= 0) {
      sceneMotion.get(image).target = reducedMotion.matches
        ? 0
        : clamp((sceneIndex - targetProgress) * sceneHeight * speed * .42, -180, 180);
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

function navigateToScene(scene) {
  const index = scenes.indexOf(scene);
  if (index < 0) return false;

  const storyTop = story.getBoundingClientRect().top + window.scrollY;
  const top = reducedMotion.matches
    ? scene.getBoundingClientRect().top + window.scrollY
    : storyTop + index * window.innerHeight;
  const hash = `#${scene.id}`;
  if (window.location.hash !== hash) window.history.pushState(null, '', hash);

  if (reducedMotion.matches) {
    window.scrollTo(0, top);
  } else {
    window.scrollTo({ top, behavior: 'smooth' });
  }
  return true;
}

document.addEventListener('click', (event) => {
  if (
    event.defaultPrevented
    || event.button !== 0
    || event.metaKey
    || event.ctrlKey
    || event.shiftKey
    || event.altKey
    || !(event.target instanceof Element)
  ) return;

  const link = event.target.closest('a[href^="#"]');
  if (!link) return;

  const id = decodeURIComponent(link.getAttribute('href').slice(1));
  const scene = scenes.find((candidate) => candidate.id === id);
  if (scene) {
    event.preventDefault();
    navigateToScene(scene);
  }
});

window.addEventListener('scroll', queueScrollUpdate, { passive: true });
window.addEventListener('resize', queueScrollUpdate);
reducedMotion.addEventListener('change', queueScrollUpdate);

window.addEventListener('pointermove', (event) => {
  if (reducedMotion.matches) return;

  pointerX = (event.clientX / window.innerWidth - .5) * 2;
  pointerY = (event.clientY / window.innerHeight - .5) * 2;
  window.requestAnimationFrame(() => {
    for (const target of depthTargets) {
      const depth = Number(target.dataset.depth);
      if (!Number.isFinite(depth)) continue;
      target.style.setProperty('--depth-x', `${pointerX * depth * 220}px`);
      target.style.setProperty('--depth-y', `${pointerY * depth * 220}px`);
    }
  });
}, { passive: true });

if (window.location.hash) {
  const initialId = decodeURIComponent(window.location.hash.slice(1));
  const initialScene = scenes.find((scene) => scene.id === initialId);
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

document.querySelectorAll('img').forEach((image) => {
  image.addEventListener('error', () => {
    console.error(`Image failed to load: ${image.getAttribute('src')}`);
  }, { once: true });
});
