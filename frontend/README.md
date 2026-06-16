# frontend

Vue 3 frontend for gopub, built with Vite, Vue Router 4, Vuex 4, Element Plus,
Axios, Sass, and Font Awesome.

## Commands

```shell
npm ci
npm run dev
npm run build
npm run preview
```

- `npm run dev` starts the Vite dev server on port `8080` and proxies `/api` to
  `http://127.0.0.1:8192`.
- `npm run build` writes assets to `../src/static/assets` and refreshes the
  Beego template at `../src/views/index.tpl`.

## Structure

```text
src/
  assets/      styles and images
  common/      API endpoint constants, storage, and helpers
  components/  shared layout and controls
  pages/       route-level views
  plugins/     app plugins
  request/     Axios setup
  router/      Vue Router routes and guards
  store/       Vuex store
```
