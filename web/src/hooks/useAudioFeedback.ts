import { useCallback, useEffect, useRef } from 'react';

type BrowserWindow = Window & typeof globalThis & {
  webkitAudioContext?: typeof AudioContext;
};

function createAudioContext() {
  const AudioCtor = window.AudioContext || (window as BrowserWindow).webkitAudioContext;
  return AudioCtor ? new AudioCtor() : null;
}

export function useAudioFeedback() {
  const ctxRef = useRef<AudioContext | null>(null);
  const lastTickRef = useRef(0);

  const unlock = useCallback(async () => {
    try {
      if (!ctxRef.current) {
        ctxRef.current = createAudioContext();
      }
      if (ctxRef.current?.state === 'suspended') {
        await ctxRef.current.resume();
      }
    } catch {
      // Audio feedback is optional; browsers may block it until interaction.
    }
  }, []);

  useEffect(() => {
    const handleInteraction = () => void unlock();
    window.addEventListener('pointerdown', handleInteraction, { passive: true });
    window.addEventListener('keydown', handleInteraction);
    return () => {
      window.removeEventListener('pointerdown', handleInteraction);
      window.removeEventListener('keydown', handleInteraction);
    };
  }, [unlock]);

  const playTick = useCallback(() => {
    const nowMs = window.performance.now();
    if (nowMs - lastTickRef.current < 35) return;
    lastTickRef.current = nowMs;

    void unlock().then(() => {
      const ctx = ctxRef.current;
      if (!ctx) return;
      const osc = ctx.createOscillator();
      const gain = ctx.createGain();
      const now = ctx.currentTime;

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
      const osc = ctx.createOscillator();
      const gain = ctx.createGain();
      const now = ctx.currentTime;

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
