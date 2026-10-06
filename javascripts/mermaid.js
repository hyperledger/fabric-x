// SPDX-License-Identifier: Apache-2.0
const mermaidZoomState = {
  scale: 1,
  x: 0,
  y: 0,
  dragging: false,
  dragStartX: 0,
  dragStartY: 0,
  dragOriginX: 0,
  dragOriginY: 0,
};

const MERMAID_ZOOM_MIN = 0.5;
const MERMAID_ZOOM_MAX = 4;
const MERMAID_ZOOM_STEP = 0.25;

function clampMermaidZoom(scale) {
  return Math.min(MERMAID_ZOOM_MAX, Math.max(MERMAID_ZOOM_MIN, scale));
}

function getMermaidLightboxSvg() {
  const lightbox = document.getElementById("mermaid-lightbox");
  if (!lightbox) {
    return null;
  }

  const content = lightbox.querySelector(".mermaid-lightbox__content");
  return content ? content.querySelector("svg") : null;
}

function updateMermaidZoom() {
  const lightbox = document.getElementById("mermaid-lightbox");
  const svg = getMermaidLightboxSvg();
  if (!lightbox || !svg) {
    return;
  }

  svg.style.transform = `translate(${mermaidZoomState.x}px, ${mermaidZoomState.y}px) scale(${mermaidZoomState.scale})`;
  svg.style.transformOrigin = "center center";

  const zoomLevel = lightbox.querySelector(".mermaid-lightbox__zoom-level");
  if (zoomLevel) {
    zoomLevel.textContent = `${Math.round(mermaidZoomState.scale * 100)}%`;
  }
}

function resetMermaidZoom() {
  mermaidZoomState.scale = 1;
  mermaidZoomState.x = 0;
  mermaidZoomState.y = 0;
  updateMermaidZoom();
}

function changeMermaidZoom(delta) {
  mermaidZoomState.scale = clampMermaidZoom(mermaidZoomState.scale + delta);
  updateMermaidZoom();
}

function zoomMermaidAtPoint(nextScale, clientX, clientY) {
  const lightbox = document.getElementById("mermaid-lightbox");
  const viewport = lightbox && lightbox.querySelector(".mermaid-lightbox__viewport");
  if (!viewport) {
    mermaidZoomState.scale = clampMermaidZoom(nextScale);
    updateMermaidZoom();
    return;
  }

  const previousScale = mermaidZoomState.scale;
  const clampedScale = clampMermaidZoom(nextScale);
  if (clampedScale === previousScale) {
    return;
  }

  const rect = viewport.getBoundingClientRect();
  const offsetX = clientX - rect.left - rect.width / 2;
  const offsetY = clientY - rect.top - rect.height / 2;
  const zoomRatio = clampedScale / previousScale;

  mermaidZoomState.x = offsetX - (offsetX - mermaidZoomState.x) * zoomRatio;
  mermaidZoomState.y = offsetY - (offsetY - mermaidZoomState.y) * zoomRatio;
  mermaidZoomState.scale = clampedScale;
  updateMermaidZoom();
}

function ensureMermaidLightbox() {
  if (document.getElementById("mermaid-lightbox")) {
    return;
  }

  const lightbox = document.createElement("div");
  lightbox.id = "mermaid-lightbox";
  lightbox.className = "mermaid-lightbox";
  lightbox.setAttribute("aria-hidden", "true");
  lightbox.innerHTML = `
    <div class="mermaid-lightbox__backdrop" data-mermaid-close></div>
    <div class="mermaid-lightbox__dialog" role="dialog" aria-modal="true" aria-label="Expanded Mermaid diagram">
      <button class="mermaid-lightbox__close" type="button" data-mermaid-close aria-label="Close diagram">×</button>
      <div class="mermaid-lightbox__toolbar" aria-label="Diagram zoom controls">
        <button class="mermaid-lightbox__zoom-button" type="button" data-mermaid-zoom-out aria-label="Zoom out">−</button>
        <span class="mermaid-lightbox__zoom-level" aria-live="polite">100%</span>
        <button class="mermaid-lightbox__zoom-button" type="button" data-mermaid-zoom-in aria-label="Zoom in">+</button>
        <button class="mermaid-lightbox__reset" type="button" data-mermaid-reset>Reset</button>
      </div>
      <div class="mermaid-lightbox__viewport">
        <div class="mermaid-lightbox__content"></div>
      </div>
      <div class="mermaid-lightbox__hint">Use + / − or mouse wheel to zoom. Drag to pan. Press Esc or click × to close.</div>
    </div>
  `;

  lightbox.addEventListener("click", function (event) {
    if (event.target.hasAttribute("data-mermaid-close")) {
      closeMermaidLightbox();
    }
  });

  lightbox.querySelector("[data-mermaid-zoom-in]").addEventListener("click", function () {
    changeMermaidZoom(MERMAID_ZOOM_STEP);
  });

  lightbox.querySelector("[data-mermaid-zoom-out]").addEventListener("click", function () {
    changeMermaidZoom(-MERMAID_ZOOM_STEP);
  });

  lightbox.querySelector("[data-mermaid-reset]").addEventListener("click", resetMermaidZoom);

  const viewport = lightbox.querySelector(".mermaid-lightbox__viewport");
  viewport.addEventListener("wheel", function (event) {
    event.preventDefault();
    const direction = event.deltaY < 0 ? MERMAID_ZOOM_STEP : -MERMAID_ZOOM_STEP;
    zoomMermaidAtPoint(mermaidZoomState.scale + direction, event.clientX, event.clientY);
  }, { passive: false });

  viewport.addEventListener("pointerdown", function (event) {
    if (event.button !== 0) {
      return;
    }

    mermaidZoomState.dragging = true;
    mermaidZoomState.dragStartX = event.clientX;
    mermaidZoomState.dragStartY = event.clientY;
    mermaidZoomState.dragOriginX = mermaidZoomState.x;
    mermaidZoomState.dragOriginY = mermaidZoomState.y;
    viewport.classList.add("mermaid-lightbox__viewport--dragging");
    if (viewport.setPointerCapture) {
      viewport.setPointerCapture(event.pointerId);
    }
  });

  viewport.addEventListener("pointermove", function (event) {
    if (!mermaidZoomState.dragging) {
      return;
    }

    mermaidZoomState.x = mermaidZoomState.dragOriginX + event.clientX - mermaidZoomState.dragStartX;
    mermaidZoomState.y = mermaidZoomState.dragOriginY + event.clientY - mermaidZoomState.dragStartY;
    updateMermaidZoom();
  });

  function stopDragging(event) {
    if (!mermaidZoomState.dragging) {
      return;
    }

    mermaidZoomState.dragging = false;
    viewport.classList.remove("mermaid-lightbox__viewport--dragging");
    if (viewport.releasePointerCapture && event && event.pointerId !== undefined) {
      viewport.releasePointerCapture(event.pointerId);
    }
  }

  viewport.addEventListener("pointerup", stopDragging);
  viewport.addEventListener("pointercancel", stopDragging);
  viewport.addEventListener("pointerleave", stopDragging);

  document.addEventListener("keydown", function (event) {
    if (event.key === "Escape") {
      closeMermaidLightbox();
    }
  });

  document.body.appendChild(lightbox);
}

function openMermaidLightbox(svg) {
  ensureMermaidLightbox();

  const lightbox = document.getElementById("mermaid-lightbox");
  const content = lightbox.querySelector(".mermaid-lightbox__content");
  const clone = svg.cloneNode(true);

  clone.setAttribute("aria-hidden", "true");

  content.replaceChildren(clone);
  lightbox.classList.add("mermaid-lightbox--open");
  lightbox.setAttribute("aria-hidden", "false");
  document.body.classList.add("mermaid-lightbox-open");
  resetMermaidZoom();
  lightbox.querySelector(".mermaid-lightbox__close").focus();
}

function closeMermaidLightbox() {
  const lightbox = document.getElementById("mermaid-lightbox");
  if (!lightbox) {
    return;
  }

  lightbox.classList.remove("mermaid-lightbox--open");
  lightbox.setAttribute("aria-hidden", "true");
  document.body.classList.remove("mermaid-lightbox-open");
  lightbox.querySelector(".mermaid-lightbox__content").replaceChildren();
  resetMermaidZoom();
}

function enhanceMermaidCharts() {
  document.querySelectorAll(".mermaid svg").forEach(function (svg) {
    const container = svg.closest(".mermaid");
    if (!container || container.dataset.zoomReady === "true") {
      return;
    }

    container.dataset.zoomReady = "true";
    container.classList.add("mermaid--zoomable");
    container.setAttribute("tabindex", "0");
    container.setAttribute("role", "button");
    container.setAttribute("aria-label", "Open diagram in larger view");
    container.setAttribute("title", "Click to enlarge diagram");

    container.addEventListener("click", function () {
      openMermaidLightbox(svg);
    });

    container.addEventListener("keydown", function (event) {
      if (event.key === "Enter" || event.key === " ") {
        event.preventDefault();
        openMermaidLightbox(svg);
      }
    });
  });
}

function initializeMermaidCharts() {
  if (typeof mermaid === "undefined") {
    return;
  }

  mermaid.initialize({
    startOnLoad: false,
    theme: document.body.getAttribute("data-md-color-scheme") === "slate" ? "dark" : "default",
    flowchart: {
      htmlLabels: true,
      nodeSpacing: 50,
      rankSpacing: 70,
      padding: 16,
      useMaxWidth: true,
    },
  });

  mermaid.run({ querySelector: ".mermaid" }).then(enhanceMermaidCharts);
}

if (typeof document$ !== "undefined" && document$.subscribe) {
  document$.subscribe(initializeMermaidCharts);
} else if (document.readyState === "loading") {
  document.addEventListener("DOMContentLoaded", initializeMermaidCharts);
} else {
  initializeMermaidCharts();
}
