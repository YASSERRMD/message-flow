import { useState } from "react";
import { extractMedia, shouldHidePlaceholderText } from "./messageUtils.js";
import { getAvatarStyle } from "./avatar.js";

function MediaContent({ media, mediaUrl }) {
  const [imgError, setImgError] = useState(false);

  // If we have a downloadable media URL, render the actual media
  if (mediaUrl && !imgError) {
    switch (media.media_type) {
      case "image":
        return (
          <img
            src={mediaUrl}
            alt={media.caption || "Image"}
            className="media-img"
            loading="lazy"
            onError={() => setImgError(true)}
          />
        );
      case "sticker":
        return (
          <img
            src={mediaUrl}
            alt="Sticker"
            className="media-img media-sticker"
            loading="lazy"
            onError={() => setImgError(true)}
          />
        );
      case "video":
        return (
          <video controls preload="metadata" className="media-video">
            <source src={mediaUrl} type={media.mime_type || "video/mp4"} />
            Your browser does not support video.
          </video>
        );
      case "audio":
        return (
          <audio controls preload="metadata" className="media-audio">
            <source src={mediaUrl} type={media.mime_type || "audio/ogg"} />
            Your browser does not support audio.
          </audio>
        );
      case "document":
        return (
          <a
            href={mediaUrl}
            target="_blank"
            rel="noopener noreferrer"
            className="media-doc-link"
            download={media.file_name || "document"}
          >
            <span className="media-icon">📄</span>
            <span className="media-label">{media.file_name || "Document"}</span>
            <span className="media-download-icon">⬇</span>
          </a>
        );
      default:
        break;
    }
  }

  // Fallback: emoji placeholders
  const icons = { image: "📷", video: "🎬", audio: "🎵", document: "📄", sticker: "🎭" };
  const labels = { image: "Image", video: "Video", audio: "Audio", document: media.file_name || "Document", sticker: "Sticker" };

  let extra = "";
  if ((media.media_type === "video" || media.media_type === "audio") && media.duration_seconds) {
    const m = Math.floor(media.duration_seconds / 60);
    const s = String(media.duration_seconds % 60).padStart(2, "0");
    extra = ` (${m}:${s})`;
  }

  return (
    <div className="media-placeholder">
      <span className="media-icon">{icons[media.media_type] || "📎"}</span>
      <span className="media-label">{labels[media.media_type] || media.media_type}{extra}</span>
    </div>
  );
}

export default function ChatMessage({ message, isGroup, formatTime, token, apiBase }) {
  const media = extractMedia(message.metadata_json);
  const hideText = shouldHidePlaceholderText(message.content);
  const senderName = message.sender_name || (message.sender || "").split("@")[0] || "Unknown";
  const isSenderUnknown = senderName === "Unknown" || senderName === "" || senderName === "?";

  // Construct the media URL if a media_path is available
  let mediaUrl = null;
  if (media?.media_path && apiBase && token) {
    mediaUrl = `${apiBase}/messages/${message.id}/media?token=${encodeURIComponent(token)}`;
  }

  return (
    <div className={`message ${message.is_outbound ? "outbound" : ""}`}>
      <div className="message-bubble">
        {!message.is_outbound && isGroup && !isSenderUnknown && (
          <div
            className="message-sender"
            style={{
              color: getAvatarStyle(senderName).color,
              fontSize: "12px",
              fontWeight: 600,
              marginBottom: "4px"
            }}
          >
            {senderName}
          </div>
        )}

        {media?.has_media && (
          <div className={`wa-media wa-media-${media.media_type}`}>
            <MediaContent media={media} mediaUrl={mediaUrl} />
          </div>
        )}

        {!hideText && (
          <div className="message-text">{message.content}</div>
        )}

        <div className="message-meta">
          <span className="message-time">{formatTime(message.timestamp || message.created_at)}</span>
          {message.is_outbound && (
            <span className="message-status">
              <i className="fas fa-check-double" style={{ color: "#53bdeb" }}></i>
            </span>
          )}
        </div>
      </div>
    </div>
  );
}
