import { useEffect, useState } from "react";

const parseValue = (raw, fallback) => {
  if (raw === null || raw === undefined) return fallback;
  try {
    return JSON.parse(raw);
  } catch (error) {
    return fallback;
  }
};

export default function useStoredState(key, initialValue) {
  const [state, setState] = useState(() => {
    return parseValue(localStorage.getItem(key), initialValue);
  });

  useEffect(() => {
    localStorage.setItem(key, JSON.stringify(state));
    window.dispatchEvent(new CustomEvent("mf-storage", { detail: { key, value: state } }));
  }, [key, state]);

  useEffect(() => {
    const handleCustom = (event) => {
      if (event.detail?.key === key) {
        setState(event.detail.value);
      }
    };
    const handleStorage = (event) => {
      if (event.key === key) {
        setState(parseValue(event.newValue, initialValue));
      }
    };
    window.addEventListener("mf-storage", handleCustom);
    window.addEventListener("storage", handleStorage);
    return () => {
      window.removeEventListener("mf-storage", handleCustom);
      window.removeEventListener("storage", handleStorage);
    };
  }, [key, initialValue]);

  return [state, setState];
}
