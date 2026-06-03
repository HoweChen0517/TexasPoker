import { useCallback, useRef } from 'react';

export function useHapticFeedback() {
  const lastVibrateRef = useRef(0);

  const tick = useCallback(() => {
    if (typeof navigator === 'undefined' || typeof navigator.vibrate !== 'function') return;
    const now = window.performance.now();
    if (now - lastVibrateRef.current < 45) return;
    lastVibrateRef.current = now;
    navigator.vibrate(12);
  }, []);

  return { tick };
}
