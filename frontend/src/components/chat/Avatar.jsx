import { useState } from "react";
import { getAvatarStyle, getInitials } from "./avatar.js";

export default function Avatar({ src, name, className }) {
  const [errored, setErrored] = useState(false);
  const style = errored ? getAvatarStyle(name) : {};

  return (
    <div className={className} style={style}>
      {!errored && src ? (
        <img
          src={src}
          alt={name || "Avatar"}
          className="avatar-img"
          onError={() => setErrored(true)}
        />
      ) : (
        getInitials(name)
      )}
    </div>
  );
}

