export const qs = (selector) => document.querySelector(selector);
export const qsa = (selector) => document.querySelectorAll(selector);

export function escapeName(name) {
  return encodeURIComponent(name).replace(/%2F/g, "/");
}

export function readableName(name) {
  if (!name) return "Unknown";
  const clean = name.split("/").pop().replace(/\.[^.]+$/, "").replace(/[-_]+/g, " ");
  try {
    return decodeURIComponent(clean);
  } catch (_) {
    return clean;
  }
}
