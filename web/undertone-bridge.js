// Spike: boots undertone's page bridge (cmd/undertone-bridge in
// undertone) beside the game. The game runs in a Web Worker and sends
// audio commands with js/emit; the bridge plays them on this thread, where an
// AudioContext exists. Added to the page by tools/build-web-audio.sh.
//
// The bridge wasm is larger than the game and can finish loading after the
// game's first commands (the music gain and the opening score), so commands
// are buffered from page load and replayed once the bridge listens.
//
// hold was added first, so it runs before the bridge's own listener for every
// event. Once the bridge exists, the first event through flushes the buffer
// ahead of itself: everything plays once, in order. The poll below flushes if
// no event comes.
(() => {
  const early = [];
  const flush = () => {
    removeEventListener('undertone', hold);
    for (const detail of early.splice(0)) dispatchEvent(new CustomEvent('undertone', { detail }));
  };
  const hold = e => { if (window.undertoneBridge) flush(); else early.push(e.detail); };
  addEventListener('undertone', hold);
  (async () => {
    await new Promise((ok, fail) => {
      const s = document.createElement('script');
      s.src = 'undertone-wasm_exec.js';
      s.onload = ok;
      s.onerror = () => fail(new Error('could not load undertone-wasm_exec.js'));
      document.head.appendChild(s);
    });
    const go = new Go();
    const { instance } = await WebAssembly.instantiateStreaming(fetch('undertone-bridge.wasm'), go.importObject);
    go.run(instance);
    while (!window.undertoneBridge) await new Promise(r => setTimeout(r, 20));
    flush();
  })().catch(err => console.error('undertone-bridge:', err));
})();
