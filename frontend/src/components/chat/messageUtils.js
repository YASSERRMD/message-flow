export function parseMetadata(metadataJson) {
  if (!metadataJson) return null;
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

