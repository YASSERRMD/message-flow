export function parseMetadata(metadataJson) {
  if (!metadataJson) return null;
  if (typeof metadataJson === "object") return metadataJson;
  try {
    return JSON.parse(metadataJson);
  } catch {
    return null;
  }
}

export function extractMedia(metadataJson) {
  const meta = parseMetadata(metadataJson);
  const media = meta?.media;
  if (!media || !media.has_media) return null;
  return media;
}

export function shouldHidePlaceholderText(content) {
  if (!content) return true;
  return /^\[(image|video|audio|document|sticker)\]$/i.test(String(content).trim());
}

const MEDIA_LABELS = {
  image: "Image",
  video: "Video",
  audio: "Audio",
  document: "Document",
  sticker: "Sticker"
};

const MEDIA_ICONS = {
  image: "fa-image",
  video: "fa-film",
  audio: "fa-wave-square",
  document: "fa-file-lines",
  sticker: "fa-note-sticky"
};

export function getMediaLabel(type) {
  return MEDIA_LABELS[type] || "Attachment";
}

export function getMediaIcon(type) {
  return MEDIA_ICONS[type] || "fa-paperclip";
}

export function getMediaUrl(message, apiBase, token) {
  const media = extractMedia(message?.metadata_json);
  if (!message?.id || !media?.media_path || !apiBase || !token) return "";
  return `${apiBase}/messages/${message.id}/media?token=${encodeURIComponent(token)}`;
}

export function formatMediaSize(bytes) {
  const size = Number(bytes);
  if (!Number.isFinite(size) || size <= 0) return "";
  const units = ["B", "KB", "MB", "GB", "TB"];
  let value = size;
  let index = 0;
  while (value >= 1024 && index < units.length - 1) {
    value /= 1024;
    index += 1;
  }
  return `${value >= 10 || index === 0 ? Math.round(value) : value.toFixed(1)} ${units[index]}`;
}

export function formatMediaDuration(seconds) {
  const total = Number(seconds);
  if (!Number.isFinite(total) || total <= 0) return "";
  const hrs = Math.floor(total / 3600);
  const mins = Math.floor((total % 3600) / 60);
  const secs = Math.floor(total % 60);
  if (hrs > 0) {
    return `${hrs}:${String(mins).padStart(2, "0")}:${String(secs).padStart(2, "0")}`;
  }
  return `${mins}:${String(secs).padStart(2, "0")}`;
}

export function isVisualMedia(type) {
  return type === "image" || type === "sticker";
}

export function buildMediaItem(message, apiBase, token) {
  const media = extractMedia(message?.metadata_json);
  if (!message || !media?.has_media) return null;

  const type = media.media_type || "document";
  const fallbackContent = shouldHidePlaceholderText(message.content) ? "" : (message.content || "");
  const caption = media.caption || fallbackContent;

  return {
    id: message.id,
    message,
    media,
    type,
    label: getMediaLabel(type),
    icon: getMediaIcon(type),
    url: getMediaUrl(message, apiBase, token),
    fileName: media.file_name || `${getMediaLabel(type)} attachment`,
    caption,
    sizeLabel: formatMediaSize(media.file_size),
    durationLabel: formatMediaDuration(media.duration_seconds),
    timestamp: message.timestamp || message.created_at || "",
    senderName: message.is_outbound ? "You" : message.sender_name || (message.sender || "").split("@")[0] || "Unknown"
  };
}
