import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import './SpinDial.css';

type SpinDialProps = {
  value: number;
  min: number;
  max: number;
  step?: number;
  disabled?: boolean;
  snapValue?: (value: number) => number;
  onChange: (value: number) => void;
  onSnap?: (value: number) => void;
  onTick?: () => void;
};

const PX_PER_STEP = 12;
const FRICTION = 0.95;
const STOP_VELOCITY = 0.035;

function clamp(value: number, min: number, max: number) {
  if (max < min) return min;
  return Math.min(max, Math.max(min, value));
}

function roundToStep(value: number, step: number) {
  return Math.round(value / step) * step;
}

export function SpinDial({
  value,
  min,
  max,
  step = 1,
  disabled = false,
  snapValue,
  onChange,
  onSnap,
  onTick
}: SpinDialProps) {
  const [visualValue, setVisualValue] = useState(value);
  const draggingRef = useRef(false);
  const lastXRef = useRef(0);
  const velocityRef = useRef(0);
  const rafRef = useRef<number | null>(null);
  const valueRef = useRef(value);

  const normalize = useCallback(
    (next: number) => {
      const stepped = roundToStep(next, step);
      const bounded = clamp(stepped, min, max);
      return snapValue ? clamp(snapValue(bounded), min, max) : bounded;
    },
    [max, min, snapValue, step]
  );

  const commitValue = useCallback(
    (raw: number, shouldSnap = false) => {
      const next = normalize(raw);
      setVisualValue(next);
      if (next !== valueRef.current) {
        valueRef.current = next;
        onChange(next);
        onTick?.();
      }
      if (shouldSnap) {
        onSnap?.(next);
      }
    },
    [normalize, onChange, onSnap, onTick]
  );

  const stopInertia = useCallback(() => {
    if (rafRef.current !== null) {
      window.cancelAnimationFrame(rafRef.current);
      rafRef.current = null;
    }
  }, []);

  const runInertia = useCallback(() => {
    stopInertia();
    const frame = () => {
      velocityRef.current *= FRICTION;
      if (Math.abs(velocityRef.current) < STOP_VELOCITY) {
        rafRef.current = null;
        commitValue(valueRef.current, true);
        return;
      }
      commitValue(valueRef.current + velocityRef.current);
      rafRef.current = window.requestAnimationFrame(frame);
    };
    rafRef.current = window.requestAnimationFrame(frame);
  }, [commitValue, stopInertia]);

  useEffect(() => {
    const normalized = normalize(value);
    valueRef.current = normalized;
    setVisualValue(normalized);
  }, [normalize, value]);

  useEffect(() => () => stopInertia(), [stopInertia]);

  const ticks = useMemo(() => {
    return [-3, -2, -1, 0, 1, 2, 3].map((offset) => {
      const tickValue = clamp(visualValue + offset * step, min, max);
      const depth = Math.abs(offset);
      return {
        key: `${offset}-${tickValue}`,
        value: tickValue,
        offset,
        opacity: depth === 0 ? 1 : depth === 1 ? 0.68 : depth === 2 ? 0.38 : 0.18,
        scale: depth === 0 ? 1.35 : depth === 1 ? 0.96 : depth === 2 ? 0.76 : 0.62
      };
    });
  }, [max, min, step, visualValue]);

  const nudge = (direction: -1 | 1) => {
    if (disabled) return;
    stopInertia();
    commitValue(valueRef.current + direction * step, true);
  };

  return (
    <div
      className={`spin-dial ${disabled ? 'disabled' : ''}`}
      role="spinbutton"
      aria-valuemin={min}
      aria-valuemax={max}
      aria-valuenow={visualValue}
      aria-disabled={disabled}
      onPointerDown={(event) => {
        if (disabled) return;
        stopInertia();
        draggingRef.current = true;
        lastXRef.current = event.clientX;
        velocityRef.current = 0;
        event.currentTarget.setPointerCapture(event.pointerId);
      }}
      onPointerMove={(event) => {
        if (!draggingRef.current || disabled) return;
        event.preventDefault();
        const deltaX = lastXRef.current - event.clientX;
        lastXRef.current = event.clientX;
        const deltaValue = deltaX / PX_PER_STEP;
        velocityRef.current = deltaValue;
        commitValue(valueRef.current + deltaValue);
      }}
      onPointerUp={(event) => {
        if (!draggingRef.current) return;
        draggingRef.current = false;
        event.currentTarget.releasePointerCapture(event.pointerId);
        if (Math.abs(velocityRef.current) > STOP_VELOCITY) {
          runInertia();
        } else {
          commitValue(valueRef.current, true);
        }
      }}
      onPointerCancel={() => {
        draggingRef.current = false;
        commitValue(valueRef.current, true);
      }}
      onWheel={(event) => {
        if (disabled) return;
        event.preventDefault();
        stopInertia();
        const direction = event.deltaY > 0 || event.deltaX > 0 ? 1 : -1;
        commitValue(valueRef.current + direction * step, true);
      }}
    >
      <button
        className="spin-dial-edge"
        type="button"
        aria-label="Decrease bet"
        disabled={disabled}
        onPointerDown={(event) => event.stopPropagation()}
        onClick={() => nudge(-1)}
      >
        -
      </button>
      <div className="spin-dial-window">
        <div className="spin-dial-track">
          {ticks.map((tick) => (
            <span
              className={`spin-dial-tick ${tick.offset === 0 ? 'current' : ''}`}
              key={tick.key}
              style={{
                transform: `translateX(calc(${tick.offset} * clamp(28px, 8vw, 52px))) scale(${tick.scale}) rotateX(${tick.offset * -9}deg)`,
                opacity: tick.opacity
              }}
            >
              {tick.value}
            </span>
          ))}
        </div>
      </div>
      <button
        className="spin-dial-edge"
        type="button"
        aria-label="Increase bet"
        disabled={disabled}
        onPointerDown={(event) => event.stopPropagation()}
        onClick={() => nudge(1)}
      >
        +
      </button>
    </div>
  );
}
