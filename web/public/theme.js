// Sets the dark-mode class before the first paint, so a dark page never flashes white. A separate
// file because the Content-Security-Policy allows no inline scripts. Same rule as src/lib/theme.ts.
(function () {
  var t = null;
  try {
    t = window.localStorage.getItem('dezetrein.theme');
  } catch (e) {
    // storage blocked: follow the system
  }
  var dark = t === 'dark' || (t !== 'light' && window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches);
  document.documentElement.classList.toggle('dark', dark);
})();
