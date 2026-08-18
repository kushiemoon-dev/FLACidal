import { writable, get } from 'svelte/store';

interface AudioSettings {
  enabled: boolean;
  volume: number;
}

const audioSettings = writable<AudioSettings>({
  enabled: false,
  volume: 70
});

let audioContext: AudioContext | null = null;

// Tones are synthesized with the Web Audio API oscillator, so no external audio files are needed
type SoundType = 'complete' | 'error' | 'queue-done' | 'click' | 'success';

// Lazily creates the audio context on first user interaction
function ensureAudioContext(): AudioContext | null {
  if (!audioContext) {
    try {
      audioContext = new (window.AudioContext || (window as any).webkitAudioContext)();
    } catch (e) {
      console.warn('Web Audio API is not supported in this environment');
      return null;
    }
  }
  return audioContext;
}

function playSuccessSound() {
  const ctx = ensureAudioContext();
  if (!ctx) return;

  const settings = get(audioSettings);
  if (!settings.enabled) return;

  const volume = 0.2 * (settings.volume / 100);

  const osc1 = ctx.createOscillator();
  const gain1 = ctx.createGain();
  osc1.connect(gain1);
  gain1.connect(ctx.destination);
  osc1.type = 'sine';
  osc1.frequency.value = 523.25; // C5
  gain1.gain.setValueAtTime(0, ctx.currentTime);
  gain1.gain.linearRampToValueAtTime(volume, ctx.currentTime + 0.02);
  gain1.gain.linearRampToValueAtTime(0, ctx.currentTime + 0.15);
  osc1.start(ctx.currentTime);
  osc1.stop(ctx.currentTime + 0.15);

  const osc2 = ctx.createOscillator();
  const gain2 = ctx.createGain();
  osc2.connect(gain2);
  gain2.connect(ctx.destination);
  osc2.type = 'sine';
  osc2.frequency.value = 659.25; // E5
  gain2.gain.setValueAtTime(0, ctx.currentTime + 0.1);
  gain2.gain.linearRampToValueAtTime(volume, ctx.currentTime + 0.12);
  gain2.gain.linearRampToValueAtTime(0, ctx.currentTime + 0.3);
  osc2.start(ctx.currentTime + 0.1);
  osc2.stop(ctx.currentTime + 0.3);
}

function playErrorSound() {
  const ctx = ensureAudioContext();
  if (!ctx) return;

  const settings = get(audioSettings);
  if (!settings.enabled) return;

  const volume = 0.15 * (settings.volume / 100);

  const osc = ctx.createOscillator();
  const gain = ctx.createGain();
  osc.connect(gain);
  gain.connect(ctx.destination);
  osc.type = 'sawtooth';
  osc.frequency.setValueAtTime(400, ctx.currentTime);
  osc.frequency.linearRampToValueAtTime(200, ctx.currentTime + 0.2);
  gain.gain.setValueAtTime(0, ctx.currentTime);
  gain.gain.linearRampToValueAtTime(volume, ctx.currentTime + 0.02);
  gain.gain.linearRampToValueAtTime(0, ctx.currentTime + 0.25);
  osc.start(ctx.currentTime);
  osc.stop(ctx.currentTime + 0.25);
}

function playClickSound() {
  const ctx = ensureAudioContext();
  if (!ctx) return;

  const settings = get(audioSettings);
  if (!settings.enabled) return;

  const volume = 0.1 * (settings.volume / 100);

  const osc = ctx.createOscillator();
  const gain = ctx.createGain();
  osc.connect(gain);
  gain.connect(ctx.destination);
  osc.type = 'sine';
  osc.frequency.value = 800;
  gain.gain.setValueAtTime(volume, ctx.currentTime);
  gain.gain.linearRampToValueAtTime(0, ctx.currentTime + 0.001);
  osc.start(ctx.currentTime);
  osc.stop(ctx.currentTime + 0.001);
}

function playQueueDoneSound() {
  const ctx = ensureAudioContext();
  if (!ctx) return;

  const settings = get(audioSettings);
  if (!settings.enabled) return;

  const volume = 0.15 * (settings.volume / 100);

  const frequencies = [523.25, 659.25, 783.99]; // C5, E5, G5 (C major chord)

  frequencies.forEach((freq, i) => {
    const osc = ctx.createOscillator();
    const gain = ctx.createGain();
    osc.connect(gain);
    gain.connect(ctx.destination);
    osc.type = 'sine';
    osc.frequency.value = freq;
    const startTime = ctx.currentTime + i * 0.05;
    gain.gain.setValueAtTime(0, startTime);
    gain.gain.linearRampToValueAtTime(volume, startTime + 0.05);
    gain.gain.linearRampToValueAtTime(0, startTime + 0.5);
    osc.start(startTime);
    osc.stop(startTime + 0.5);
  });
}

export function playSound(type: SoundType) {
  const settings = get(audioSettings);
  if (!settings.enabled) return;

  switch (type) {
    case 'complete':
    case 'success':
      playSuccessSound();
      break;
    case 'error':
      playErrorSound();
      break;
    case 'queue-done':
      playQueueDoneSound();
      break;
    case 'click':
      playClickSound();
      break;
  }
}

export function initializeAudioSettings(enabled: boolean, volume: number) {
  audioSettings.set({
    enabled,
    volume: volume || 70
  });
}

export function updateAudioSettings(enabled: boolean, volume: number) {
  audioSettings.set({ enabled, volume });
}

export function testSound() {
  const settings = get(audioSettings);
  const wasEnabled = settings.enabled;

  // Briefly force-enable so the preview actually plays
  audioSettings.set({ ...settings, enabled: true });
  playSuccessSound();

  // Put the enabled flag back the way it was
  if (!wasEnabled) {
    audioSettings.set({ ...settings, enabled: false });
  }
}
