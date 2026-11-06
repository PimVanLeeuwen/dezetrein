// Placeholder shell until Phase 4 builds the "Find my train" flow. Dutch first, English below;
// the full NL/EN dictionary comes with the real screens.
export default function App() {
  return (
    <main className="mx-auto flex min-h-dvh max-w-xl flex-col justify-center gap-6 px-6 py-12 font-sans text-neutral-900 dark:text-neutral-100">
      <h1 className="text-4xl font-bold tracking-tight">
        deze<span className="text-brand dark:text-brand-light">trein</span>
      </h1>
      <p className="text-xl">Vind de trein waarin je zit: volgende stations, vertraging en drukte.</p>
      <p lang="en" className="text-lg text-neutral-600 dark:text-neutral-400">
        Find the train you are on: next stations, delays and crowding.
      </p>
      <p className="text-sm font-medium uppercase tracking-wide text-brand dark:text-brand-light">
        Binnenkort <span lang="en">· Coming soon</span>
      </p>
    </main>
  );
}
