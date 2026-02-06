const AVATAR_COLORS = [
  { bg: "#f0f9ff", color: "#0369a1" },
  { bg: "#fef3c7", color: "#a16207" },
  { bg: "#f5f3ff", color: "#6b21a8" },
  { bg: "#f0fdf4", color: "#15803d" },
  { bg: "#fdf2f8", color: "#be185d" },
  { bg: "#ecfeff", color: "#0e7490" },
  { bg: "#fff7ed", color: "#c2410c" },
  { bg: "#eff6ff", color: "#1e40af" }
];

export function getInitials(name) {
  if (!name) return "?";
  return String(name)
    .split(" ")
    .filter(Boolean)
    .map((w) => w[0])
    .join("")
    .substring(0, 2)
    .toUpperCase();
}

export function getAvatarStyle(name) {
  const key = (name || "?")[0] || "?";
  const index = key.charCodeAt(0) % AVATAR_COLORS.length;
  return { background: AVATAR_COLORS[index].bg, color: AVATAR_COLORS[index].color };
}

