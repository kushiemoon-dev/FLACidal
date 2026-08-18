# Svelte + TS + Vite

Use this template as a starting point for building with Svelte and TypeScript on top of Vite.

## Recommended IDE Setup

[VS Code](https://code.visualstudio.com/)

+ [Svelte](https://marketplace.visualstudio.com/items?itemName=svelte.svelte-vscode).

## Looking for an official Svelte framework instead?

Take a look at [SvelteKit](https://github.com/sveltejs/kit#readme) — it's also built on Vite. Its serverless-first design deploys anywhere and adapts across platforms, with TypeScript, SCSS, and Less supported out of the box, plus straightforward add-ons for mdsvex, GraphQL, PostCSS, Tailwind CSS, and more.

## Technical considerations

**Why pick this instead of SvelteKit?**

- SvelteKit ships its own routing approach, which isn't always what people want.
- SvelteKit is, first and foremost, a framework that happens to run on Vite — it isn't simply a Vite app.
  For instance, `vite dev` and `vite build` won't function inside a SvelteKit project.

This template keeps things minimal — just enough to get Vite, TypeScript, and Svelte running together — while still
caring about developer experience around HMR and editor intellisense. It matches the capabilities of the
other `create-vite` templates and works well as a first step for anyone new to Vite + Svelte.

If you eventually need the broader feature set SvelteKit offers, this template mirrors SvelteKit's structure closely
enough to make migrating straightforward.

**Why `global.d.ts` rather than `compilerOptions.types` in `jsconfig.json` or `tsconfig.json`?**

Declaring `compilerOptions.types` excludes any type not explicitly listed there. Triple-slash references, by contrast,
preserve TypeScript's default behavior of pulling type information from across the whole workspace, on top of adding
the `svelte` and `vite/client` types.

**What's `.vscode/extensions.json` for?**

Rather than just mentioning recommended extensions in the README, this file lets VS Code prompt the user to install
them directly when the project is opened.

**Why does the TS template turn on `allowJs`?**

Setting `allowJs: false` would block `.js` files in the project, sure — but JavaScript syntax inside `.svelte` files
would still slip through. It would also force `checkJs: false`, which is the worst combination: no guarantee the
codebase is fully TypeScript, and weaker type-checking on whatever JavaScript remains. There are also legitimate
scenarios where a mixed-language codebase makes sense.

**Why doesn't HMR hold onto my component's local state?**

Preserving state across HMR updates has several sharp edges, which is why it's off by default in both `svelte-hmr`
and `@sveltejs/vite-plugin-svelte`. See the details [here](https://github.com/rixo/svelte-hmr#svelte-hmr).

For state you need to survive HMR, an external store that HMR won't replace is the way to go.

```ts
// store.ts
// An extremely simple external store
import { writable } from 'svelte/store'
export default writable(0)
```
