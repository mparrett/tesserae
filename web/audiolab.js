// Spike: ?audiolab on the audio build (tools/build-web-audio.sh) puts a
// latency panel over the game. It sets the bridge's queue while playing,
// shows the queue and the underrun count, and times a trigger to the moment
// its sound leaves undertone's worklet. Without ?audiolab it does nothing.
//
// The timing taps the bridge's output: loaded before the bridge, it wraps
// AudioNode.connect so the worklet node webaudio connects to the speakers
// also feeds an analyser. The measured delay covers the event, the bridge's
// VM, the render queue and the worklet; the browser's own output latency
// (AudioContext.outputLatency, where it reports one) comes on top.
(() => {
  if (!new URLSearchParams(location.search).has('audiolab')) return;

  let analyser = null, actx = null;
  const connect = AudioNode.prototype.connect;
  AudioNode.prototype.connect = function (dest, ...rest) {
    if (!analyser && this instanceof AudioWorkletNode && dest instanceof AudioDestinationNode) {
      actx = this.context;
      analyser = actx.createAnalyser();
      analyser.fftSize = 256;
      connect.call(this, analyser);
    }
    return connect.call(this, dest, ...rest);
  };

  // The game sets the bed's level once at start; remember it to restore after a test.
  let musicGain = 0.3;
  addEventListener('undertone', (e) => {
    const m = /^\[:music-gain ([\d.]+)\]$/.exec(e.detail?.form || '');
    if (m && !testing) musicGain = parseFloat(m[1]);
  });
  const send = (form) => dispatchEvent(new CustomEvent('undertone', { detail: { form } }));

  const panel = document.createElement('div');
  panel.style.cssText = 'position:fixed;top:max(8px,env(safe-area-inset-top));right:8px;z-index:9;' +
    'background:#101018e8;color:#d8d6cf;font:12px/1.5 monospace;padding:8px 10px;border:1px solid #444;' +
    'border-radius:8px;max-width:min(320px,calc(100vw - 16px))';
  panel.innerHTML = `<b>audiolab</b> <span id="al-stat">waiting for audio…</span>
    <div id="al-q" style="margin:6px 0;display:flex;gap:4px;flex-wrap:wrap"></div>
    <button id="al-tap">tap test ×8</button> <span id="al-tap-out"></span>
    <div id="al-log" style="opacity:.75"></div>`;
  const style = 'font:inherit;color:inherit;background:#22222c;border:1px solid #555;border-radius:6px;padding:4px 8px';
  for (const ms of [40, 60, 80, 120, 200]) {
    const b = document.createElement('button');
    b.textContent = ms; b.style.cssText = style;
    b.onclick = () => log(`queue → ${window.undertoneBridge?.setQueue(ms) ?? 'not started'} ms`);
    panel.querySelector('#al-q').append(b);
  }
  panel.querySelector('#al-tap').style.cssText = style;
  addEventListener('DOMContentLoaded', () => document.body.append(panel));
  const $ = (id) => panel.querySelector(id);
  const log = (s) => { $('#al-log').textContent = s; };

  // Underruns since the last change of queue, so a setting's count starts at 0.
  let base = 0, lastQueue = null;
  setInterval(() => {
    const s = window.undertoneBridge?.stats?.();
    if (!s || !s.started) return;
    if (s.queueMs !== lastQueue) { base = s.underruns; lastQueue = s.queueMs; }
    const out = actx ? Math.round(1000 * (actx.outputLatency || actx.baseLatency || 0)) : '?';
    $('#al-stat').innerHTML = `queue ${s.queueMs || '80 (default)'} ms · queued ${s.queuedMs} ms · ` +
      `<span style="color:${s.underruns > base ? '#ff7a6a' : 'inherit'}">underruns ${s.underruns - base}</span>` +
      ` · output ${out} ms`;
  }, 250);

  // A blip with the bed muted, timed from dispatch to the first analyser read
  // well above the floor measured just before it. Polling every few ms bounds
  // the error at a poll. A sound already playing (the bed fading, a game
  // sound) would read as an early onset, so each trial waits for quiet and the
  // threshold rides on the floor; run it on the title screen.
  let testing = false;
  const sleep = (ms) => new Promise((r) => setTimeout(r, ms));
  const peak = (buf) => { analyser.getFloatTimeDomainData(buf); let p = 0; for (const v of buf) p = Math.max(p, Math.abs(v)); return p; };
  async function floor(buf, ms) {
    let f = 0;
    for (const t0 = performance.now(); performance.now() - t0 < ms; await sleep(4)) f = Math.max(f, peak(buf));
    return f;
  }
  async function quiet(buf, limit) {
    for (const t0 = performance.now(); performance.now() - t0 < limit;) if (await floor(buf, 150) < 0.01) return true;
    return false;
  }
  async function one(buf) {
    const thr = Math.max(0.03, 4 * await floor(buf, 80));
    const t0 = performance.now();
    send('[:trigger {:shot :blip :degree 4 :octave 1 :gain 0.9}]');
    while (performance.now() - t0 < 1000) {
      if (peak(buf) > thr) return performance.now() - t0;
      await sleep(2);
    }
    return null;
  }
  $('#al-tap').onclick = async () => {
    if (testing || !analyser || !window.undertoneBridge?.stats().started) { log('start the audio first (tap the game)'); return; }
    testing = true;
    $('#al-tap-out').textContent = 'running…';
    send('[:music-gain 0]');
    const buf = new Float32Array(analyser.fftSize), got = [];
    if (!await quiet(buf, 4000)) log('output never went quiet; results may read early');
    for (let i = 0; i < 8; i++) {
      await quiet(buf, 1000);                // the last blip's tail
      const d = await one(buf);
      if (d != null) got.push(d);
    }
    send(`[:music-gain ${musicGain}]`);
    testing = false;
    got.sort((a, b) => a - b);
    const med = got.length ? Math.round(got[got.length >> 1]) : null;
    const out = actx ? Math.round(1000 * (actx.outputLatency || actx.baseLatency || 0)) : 0;
    $('#al-tap-out').textContent = med == null ? 'no onset heard' :
      `median ${med} ms (${Math.round(got[0])}–${Math.round(got[got.length - 1])}, n=${got.length}) + output ${out} ms`;
  };
})();
