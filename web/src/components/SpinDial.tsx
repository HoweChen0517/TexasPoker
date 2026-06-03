import { useCallback, useEffect, useRef, useState } from 'react';
import './SpinDial.css';

type SpinDialProps = {
  value: number;
  min: number;
  max: number;
  step?: number;
  disabled?: boolean;
  normalizeValue: (value: number) => number;
  onChange: (value: number) => void;
  onSnap?: (value: number) => void;
  onTick?: () => void;
};

const PX_PER_STEP = 10;
const FRICTION = 0.95;
const STOP_VELOCITY = 0.5;

function clamp(value: number, min: number, max: number) {
  if (max < min) return max;
  return Math.min(max, Math.max(min, value));
}

function roundToStep(value: number, step: number) {
  return Math.round(value / step) * step;
}

function rubberValue(value: number, min: number, max: number) {
  if (value < min) return min + (value - min) * 0.3;
  if (value > max) return max + (value - max) * 0.3;
  return value;
}

export function SpinDial({
  value,
  min,
  max,
  step = 10,
  disabled = false,
  normalizeValue,
  onChange,
  onSnap,
  onTick
}: SpinDialProps) {
  const [visualValue, setVisualValue] = useState(value);
  const rawRef = useRef(value);
  const valueRef = useRef(value);
  const lastYRef = useRef(0);
  const velocityPxRef = useRef(0);
  const draggingRef = useRef(false);
  const rafRef = useRef<number | null>(null);

  const normalize = useCallback(
    (next: number) => {
      const stepped = roundToStep(next, step);
      return clamp(normalizeValue(stepped), 0, max);
    },
    [max, normalizeValue, step]
  );

  const stopInertia = useCallback(() => {
    if (rafRef.current !== null) {
      window.cancelAnimationFrame(rafRef.current);
      rafRef.current = null;
    }
  }, []);

  const commit = useCallback(
    (raw: number, shouldSnap = false) => {
      rawRef.current = raw;
      const rubbered = rubberValue(raw, min, max);
      const next = normalize(rubbered);
      setVisualValue(next);

      if (next !== valueRef.current) {
        valueRef.current = next;
        onChange(next);
        onTick?.();
      }

      if (shouldSnap) {
        rawRef.current = next;
        setVisualValue(next);
        onSnap?.(next);
      }
    },
    [max, min, normalize, onChange, onSnap, onTick]
  );

  const runInertia = useCallback(() => {
    stopInertia();
    const frame = () => {
      velocityPxRef.current *= FRICTION;
      if (Math.abs(velocityPxRef.current) < STOP_VELOCITY) {
        rafRef.current = null;
        commit(rawRef.current, true);
        return;
      }

      commit(rawRef.current + (velocityPxRef.current / PX_PER_STEP) * step);
      rafRef.current = window.requestAnimationFrame(frame);
    };
    rafRef.current = window.requestAnimationFrame(frame);
  }, [commit, step, stopInertia]);

  useEffect(() => {
    const next = normalize(value);
    rawRef.current = next;
    valueRef.current = next;
    setVisualValue(next);
  }, [normalize, value]);

  useEffect(() => () => stopInertia(), [stopInertia]);

  const nudge = (direction: -1 | 1) => {
    if (disabled) return;
    stopInertia();
    commit(valueRef.current + direction * step, true);
  };

  return (
    <div
      className={`spin-dial vertical ${disabled ? 'disabled' : ''}`}
      role="spinbutton"
      aria-valuemin={min}
      aria-valuemax={max}
      aria-valuenow={visualValue}
      aria-disabled={disabled}
      onPointerDown={(event) => {
        if (disabled) return;
        stopInertia();
        draggingRef.current = true;
        lastYRef.current = event.clientY;
        velocityPxRef.current = 0;
        event.currentTarget.setPointerCapture(event.pointerId);
      }}
      onPointerMove={(event) => {
        if (!draggingRef.current || disabled) return;
        event.preventDefault();
        const deltaY = event.clientY - lastYRef.current;
        lastYRef.current = event.clientY;
        velocityPxRef.current = deltaY;
        commit(rawRef.current + (deltaY / PX_PER_STEP) * step);
      }}
      onPointerUp={(event) => {
        if (!draggingRef.current) return;
        draggingRef.current = false;
        event.currentTarget.releasePointerCapture(event.pointerId);
        if (Math.abs(velocityPxRef.current) >= STOP_VELOCITY) {
          runInertia();
          return;
        }
        commit(rawRef.current, true);
      }}
      onPointerCancel={() => {
        draggingRef.current = false;
        commit(rawRef.current, true);
      }}
      onWheel={(event) => {
        if (disabled) return;
        event.preventDefault();
        stopInertia();
        const direction = event.deltaY > 0 ? 1 : -1;
        commit(valueRef.current + direction * step, true);
      }}
    >
      <button
        className="spin-dial-nudge"
        type="button"
        disabled={disabled}
        aria-label="Increase wager"
        onPointerDown={(event) => event.stopPropagation()}
        onClick={() => nudge(1)}
      >
        ▲
      </button>
      <div className="spin-dial-window">
        <div className="spin-dial-track">
          <span className="spin-dial-grip" />
          <span className="spin-dial-grip" />
          <span className="spin-dial-grip" />
          <span className="spin-dial-grip" />
        </div>
      </div>
      <button
        className="spin-dial-nudge"
        type="button"
        disabled={disabled}
        aria-label="Decrease wager"
        onPointerDown={(event) => event.stopPropagation()}
        onClick={() => nudge(-1)}
      >
        ▼
      </button>
    </div>
  );
}
