import { useCallback, useRef } from 'react';

function canVibrate() {
  return typeof navigator !== 'undefined' && typeof navigator.vibrate === 'function';
}

export function useHapticFeedback() {
  const lastVibrateRef = useRef(0);

  const tick = useCallback(() => {
    if (!canVibrate()) return;
    const now = window.performance.now();
    if (now - lastVibrateRef.current < 45) return;
    lastVibrateRef.current = now;
    navigator.vibrate(12);
  }, []);

  return { tick };
}
