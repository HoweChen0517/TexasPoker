import { useCallback, useEffect, useRef } from 'react';

type AudioWindow = Window & typeof globalThis & {
  webkitAudioContext?: typeof AudioContext;
};

function makeAudioContext() {
  const AudioCtor = window.AudioContext || (window as AudioWindow).webkitAudioContext;
  return AudioCtor ? new AudioCtor() : null;
}

export function useAudioFeedback() {
  const ctxRef = useRef<AudioContext | null>(null);
  const lastTickRef = useRef(0);

  const unlock = useCallback(async () => {
    try {
      if (!ctxRef.current) {
        ctxRef.current = makeAudioContext();
      }
      if (ctxRef.current?.state === 'suspended') {
        await ctxRef.current.resume();
      }
    } catch {
      // Browsers may block audio before a user gesture. Feedback is optional.
    }
  }, []);

  useEffect(() => {
    const handleGesture = () => void unlock();
    window.addEventListener('pointerdown', handleGesture, { passive: true });
    window.addEventListener('keydown', handleGesture);
    return () => {
      window.removeEventListener('pointerdown', handleGesture);
      window.removeEventListener('keydown', handleGesture);
    };
  }, [unlock]);

  const playTick = useCallback(() => {
    const nowMs = window.performance.now();
    if (nowMs - lastTickRef.current < 40) return;
    lastTickRef.current = nowMs;

    void unlock().then(() => {
      const ctx = ctxRef.current;
      if (!ctx) return;
      const now = ctx.currentTime;
      const osc = ctx.createOscillator();
      const gain = ctx.createGain();

      osc.type = 'sine';
      osc.frequency.setValueAtTime(820, now);
      gain.gain.setValueAtTime(0.0001, now);
      gain.gain.exponentialRampToValueAtTime(0.035, now + 0.004);
      gain.gain.exponentialRampToValueAtTime(0.0001, now + 0.05);

      osc.connect(gain);
      gain.connect(ctx.destination);
      osc.start(now);
      osc.stop(now + 0.055);
    });
  }, [unlock]);

  const playSnap = useCallback(() => {
    void unlock().then(() => {
      const ctx = ctxRef.current;
      if (!ctx) return;
      const now = ctx.currentTime;
      const osc = ctx.createOscillator();
      const gain = ctx.createGain();

      osc.type = 'sine';
      osc.frequency.setValueAtTime(1200, now);
      osc.frequency.exponentialRampToValueAtTime(600, now + 0.15);
      gain.gain.setValueAtTime(0.0001, now);
      gain.gain.exponentialRampToValueAtTime(0.055, now + 0.012);
      gain.gain.exponentialRampToValueAtTime(0.0001, now + 0.15);

      osc.connect(gain);
      gain.connect(ctx.destination);
      osc.start(now);
      osc.stop(now + 0.16);
    });
  }, [unlock]);

  return { playTick, playSnap, unlock };
}
