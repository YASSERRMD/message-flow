import { useState } from "react";
import {
  buildMediaItem,
  extractMedia,
  getMediaIcon,
  getMediaUrl,
  shouldHidePlaceholderText
} from "./messageUtils.js";
import { getAvatarStyle } from "./avatar.js";

function MediaContent({ item, onPreviewMedia }) {
  const [imgError, setImgError] = useState(false);
  const media = item.media;
  const mediaUrl = item.url;

  // If we have a downloadable media URL, render the actual media
  if (mediaUrl && !imgError) {
    switch (media.media_type) {
      case "image":
        return (
          <button type="button" className="media-visual-btn" onClick={() => onPreviewMedia?.(item.id)}>
            <img
              src={mediaUrl}
              alt={media.caption || "Image"}
              className="media-img"
              loading="lazy"
              onError={() => setImgError(true)}
            />
            <span className="media-overlay-pill"><i className="fas fa-expand"></i> Preview</span>
          </button>
        );
      case "sticker":
        return (
          <button type="button" className="media-visual-btn sticker" onClick={() => onPreviewMedia?.(item.id)}>
            <img
              src={mediaUrl}
              alt="Sticker"
              className="media-img media-sticker"
              loading="lazy"
              onError={() => setImgError(true)}
            />
            <span className="media-overlay-pill"><i className="fas fa-expand"></i> Preview</span>
          </button>
        );
      case "video":
        return (
          <div className="media-video-shell">
            <video controls preload="metadata" className="media-video">
              <source src={mediaUrl} type={media.mime_type || "video/mp4"} />
              Your browser does not support video.
            </video>
            <button type="button" className="media-inline-action" onClick={() => onPreviewMedia?.(item.id)}>
              <i className="fas fa-expand"></i>
              Expand
            </button>
          </div>
        );
      case "audio":
        return (
          <div className="media-audio-shell">
            <audio controls preload="metadata" className="media-audio">
              <source src={mediaUrl} type={media.mime_type || "audio/ogg"} />
              Your browser does not support audio.
            </audio>
            <button type="button" className="media-inline-action" onClick={() => onPreviewMedia?.(item.id)}>
              <i className="fas fa-wave-square"></i>
              Details
            </button>
          </div>
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
            <span className="media-doc-icon"><i className="fas fa-file-lines"></i></span>
            <span className="media-doc-copy">
              <span className="media-label">{media.file_name || "Document"}</span>
              <small>{media.mime_type || "Document attachment"}</small>
            </span>
            <span className="media-download-icon"><i className="fas fa-download"></i></span>
          </a>
        );
      default:
        break;
    }
  }

  // Fallback: emoji placeholders
  return (
    <div className="media-placeholder">
      <span className="media-icon"><i className={`fas ${getMediaIcon(media.media_type)}`}></i></span>
      <span className="media-label">{media.file_name || media.media_type || "Attachment"}</span>
    </div>
  );
}

function MediaToolbar({ item, onPreviewMedia }) {
  const details = [item.label, item.durationLabel, item.sizeLabel].filter(Boolean);

  return (
    <div className="media-toolbar">
      <div className="media-toolbar-meta">
        {details.map((detail) => (
          <span key={detail} className="media-toolbar-chip">{detail}</span>
        ))}
      </div>
      <div className="media-toolbar-actions">
        {item.url && (
          <button type="button" className="media-toolbar-btn" onClick={() => onPreviewMedia?.(item.id)}>
            <i className="fas fa-expand"></i>
            Open
          </button>
        )}
        {item.url && (
          <a href={item.url} download={item.fileName} className="media-toolbar-btn">
            <i className="fas fa-download"></i>
            Save
          </a>
        )}
      </div>
    </div>
  );
}

export default function ChatMessage({ message, isGroup, formatTime, token, apiBase, onPreviewMedia, isHighlighted = false }) {
  const media = extractMedia(message.metadata_json);
  const hideText = shouldHidePlaceholderText(message.content);
  const senderName = message.sender_name || (message.sender || "").split("@")[0] || "Unknown";
  const isSenderUnknown = senderName === "Unknown" || senderName === "" || senderName === "?";
  const mediaItem = buildMediaItem(message, apiBase, token);
  const mediaUrl = getMediaUrl(message, apiBase, token);

  return (
    <div id={`message-${message.id}`} className={`message ${message.is_outbound ? "outbound" : ""} ${isHighlighted ? "highlighted" : ""}`}>
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
            {mediaItem ? <MediaContent item={mediaItem} onPreviewMedia={onPreviewMedia} /> : null}
            {mediaItem && mediaUrl ? <MediaToolbar item={mediaItem} onPreviewMedia={onPreviewMedia} /> : null}
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
