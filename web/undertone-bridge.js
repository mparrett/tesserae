// Spike: boots undertone's page bridge (cmd/undertone-bridge in
// let-go-undertone) beside the game. The game runs in a Web Worker and sends
// audio commands with js/emit; the bridge plays them on this thread, where an
// AudioContext exists. Added to the page by tools/build-web-audio.sh.
//
// The bridge wasm is larger than the game and can finish loading after the
// game's first commands (the music gain and the opening score), so commands
// are buffered from page load and replayed once the bridge listens.
(() => {
  const early = [];
  const hold = e => early.push(e.detail);
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
    removeEventListener('undertone', hold);
    for (const detail of early) dispatchEvent(new CustomEvent('undertone', { detail }));
  })().catch(err => console.error('undertone-bridge:', err));
})();
