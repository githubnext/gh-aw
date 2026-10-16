import { initializeSlideshowDrawing } from "./slideshow-drawing";
import { initializeSlideshowSnippets } from "./slideshow-snippets";

export interface SlideshowController {
  open(): void;
  destroy(): void;
}

export function createSlideshow(trigger: HTMLButtonElement): SlideshowController {
  const dialog = document.querySelector<HTMLDialogElement>("#landing-slideshow");
  const deck = document.querySelector<HTMLElement>("[data-slideshow-slides]");
  const viewport = dialog?.querySelector<HTMLElement>("[data-slideshow-viewport]");
  const stage = dialog?.querySelector<HTMLElement>("[data-slideshow-stage]");
  const ink = dialog?.querySelector<SVGSVGElement>("[data-slideshow-ink]");
  const previous = dialog?.querySelector<HTMLButtonElement>("[data-slideshow-previous]");
  const next = dialog?.querySelector<HTMLButtonElement>("[data-slideshow-next]");
  const close = dialog?.querySelector<HTMLButtonElement>("[data-slideshow-close]");
  const status = dialog?.querySelector<HTMLElement>("[data-slideshow-status]");
  if (!dialog || !deck || !viewport || !stage || !ink || !previous || !next || !close || !status) {
    throw new Error("The slideshow markup is incomplete.");
  }
  const slides = Array.from(deck.querySelectorAll<HTMLElement>(":scope > section:not([data-slideshow-hide])"));
  if (!slides.length) throw new Error("No presentation slides are available.");

  const controller = new AbortController();
  const { signal } = controller;
  const desktop = window.matchMedia("(min-width: 50rem)");
  const reducedMotion = window.matchMedia("(prefers-reduced-motion: reduce)");
  const placeholder = document.createComment("Landing slideshow");
  const cleanupDrawing = initializeSlideshowDrawing();
  const cleanupSnippets = initializeSlideshowSnippets();
  let index = 0;
  let requestedIndex = 0;
  let scrollY = 0;
  let transition: ViewTransition | undefined;
  let entrances: Animation[] = [];

  const cancelEntrances = () => {
    entrances.forEach(animation => animation.cancel());
    entrances = [];
  };

  const animateEntrance = (slide: HTMLElement, delay: number) => {
    if (reducedMotion.matches) return;
    const elements: HTMLElement[] = [];
    const collect = (parent: HTMLElement) => {
      for (const element of parent.children) {
        if (!(element instanceof HTMLElement) || !element.checkVisibility() || element.matches('[aria-hidden="true"], script, style')) continue;
        if (element.matches(".aw-container, .aw-split, .aw-split-main, .aw-hero-inner, .aw-hero-text, [data-variant-group], [data-variant], ul, ol")) {
          collect(element);
        } else {
          elements.push(element);
        }
      }
    };
    collect(slide);
    const duration = elements.length === 1 ? 1000 - delay : 350;
    entrances = elements.map((element, i) => {
      const stagger = elements.length > 1 ? (i * (1000 - delay - duration)) / (elements.length - 1) : 0;
      const animation = element.animate(
        [
          { opacity: 0, transform: "translateY(16px)" },
          { opacity: 1, transform: "translateY(0)" },
        ],
        { duration, delay: delay + stagger, easing: "cubic-bezier(0.2, 0.7, 0.2, 1)", fill: "backwards" }
      );
      animation.id = "aw-slide-entrance";
      return animation;
    });
  };

  const fit = () => {
    if (!dialog.open) return;
    deck.style.removeProperty("--aw-slide-width");
    deck.style.removeProperty("--aw-slide-height");
    const width = deck.offsetWidth;
    const height = deck.offsetHeight;
    if (!width || !height) return;
    const scale = Math.min(viewport.clientWidth / width, viewport.clientHeight / height);
    const slideWidth = viewport.clientWidth / scale;
    const slideHeight = viewport.clientHeight / scale;
    deck.style.setProperty("--aw-slide-width", `${slideWidth}px`);
    deck.style.setProperty("--aw-slide-height", `${slideHeight}px`);
    deck.style.transform = `scale(${scale})`;
    ink.setAttribute("viewBox", `0 0 ${slideWidth} ${slideHeight}`);
  };

  const observer = new ResizeObserver(fit);
  observer.observe(viewport);
  observer.observe(deck);

  const pauseVideos = (slide: HTMLElement) => {
    slide.querySelectorAll("video").forEach(video => video.pause());
  };

  const showSlide = (target: number, entranceDelay = 0) => {
    cancelEntrances();
    const newIndex = Math.max(0, Math.min(slides.length - 1, target));
    if (newIndex !== index) pauseVideos(slides[index]);
    index = newIndex;
    slides.forEach((slide, i) => {
      slide.hidden = i !== index;
    });
    previous.disabled = index === 0;
    next.disabled = index === slides.length - 1;
    const heading = Array.from(slides[index].querySelectorAll<HTMLElement>("h1, h2")).find(element => element.checkVisibility());
    status.textContent = `${index + 1} / ${slides.length} — ${heading?.textContent?.trim() ?? "Slide"}`;
    // Keep focus out of hidden slides and disabled navigation buttons.
    if (document.activeElement instanceof HTMLElement && (document.activeElement.closest("section[hidden]") || (document.activeElement === previous && previous.disabled) || (document.activeElement === next && next.disabled))) {
      close.focus({ preventScroll: true });
    }
    fit();
    dialog.dataset.slideIndex = String(index);
    dialog.dispatchEvent(new Event("slideshow:slide-change"));
    animateEntrance(slides[index], entranceDelay);
  };

  const navigate = (target: number) => {
    const newIndex = Math.max(0, Math.min(slides.length - 1, target));
    if (newIndex === requestedIndex) return;
    const direction = newIndex > requestedIndex ? "forward" : "backward";
    requestedIndex = newIndex;
    transition?.skipTransition();
    if (!document.startViewTransition || reducedMotion.matches) {
      showSlide(newIndex);
      return;
    }
    document.documentElement.dataset.slideshowDirection = direction;
    const current = document.startViewTransition(() => {
      // Reserve the frame transition's 280ms within the one-second entrance.
      if (dialog.open && requestedIndex === newIndex) showSlide(newIndex, 280);
    });
    transition = current;
    current.finished.then(() => {
      if (transition !== current) return;
      transition = undefined;
      delete document.documentElement.dataset.slideshowDirection;
    });
  };

  const restore = () => {
    cancelEntrances();
    transition?.skipTransition();
    transition = undefined;
    delete document.documentElement.dataset.slideshowDirection;
    if (!placeholder.parentNode) return;
    slides.forEach(slide => {
      pauseVideos(slide);
      slide.hidden = false;
    });
    deck.style.removeProperty("transform");
    deck.style.removeProperty("--aw-slide-width");
    deck.style.removeProperty("--aw-slide-height");
    placeholder.replaceWith(deck);
    dialog.dispatchEvent(new Event("slideshow:reset"));
    window.scrollTo({ top: scrollY, behavior: "instant" });
    trigger.focus({ preventScroll: true });
  };

  const open = () => {
    if (!desktop.matches || dialog.open) return;
    scrollY = window.scrollY;
    deck.replaceWith(placeholder);
    stage.prepend(deck);
    dialog.showModal();
    requestedIndex = 0;
    showSlide(0);
    close.focus({ preventScroll: true });
  };

  previous.addEventListener("click", () => navigate(requestedIndex - 1), { signal });
  next.addEventListener("click", () => navigate(requestedIndex + 1), { signal });
  close.addEventListener(
    "click",
    () => {
      dialog.close();
      restore();
    },
    { signal }
  );
  dialog.addEventListener(
    "cancel",
    event => {
      event.preventDefault();
      dialog.close();
      restore();
    },
    { signal }
  );
  dialog.addEventListener(
    "close",
    () => {
      if (!dialog.open) restore();
    },
    { signal }
  );
  desktop.addEventListener(
    "change",
    () => {
      if (!desktop.matches && dialog.open) {
        dialog.close();
        restore();
      }
    },
    { signal }
  );
  reducedMotion.addEventListener(
    "change",
    () => {
      if (reducedMotion.matches) {
        cancelEntrances();
        transition?.skipTransition();
      }
    },
    { signal }
  );
  dialog.addEventListener(
    "keydown",
    event => {
      if (dialog.querySelector("dialog[open]")) return;
      if (event.defaultPrevented || event.altKey || event.ctrlKey || event.metaKey || (event.target instanceof HTMLElement && event.target.closest("input, textarea, select, video, summary, [contenteditable]"))) return;
      if (event.target instanceof HTMLElement && event.target.closest('[role="tab"]') && ["ArrowLeft", "ArrowRight", "Home", "End"].includes(event.key)) return;
      const destinations: Record<string, number> = {
        ArrowRight: requestedIndex + 1,
        ArrowDown: requestedIndex + 1,
        PageDown: requestedIndex + 1,
        ArrowLeft: requestedIndex - 1,
        ArrowUp: requestedIndex - 1,
        PageUp: requestedIndex - 1,
        Home: 0,
        End: slides.length - 1,
      };
      if (!(event.key in destinations)) return;
      event.preventDefault();
      navigate(destinations[event.key]);
    },
    { signal }
  );

  return {
    open,
    destroy() {
      controller.abort();
      observer.disconnect();
      if (dialog.open) dialog.close();
      restore();
      cleanupDrawing();
      cleanupSnippets();
    },
  };
}
