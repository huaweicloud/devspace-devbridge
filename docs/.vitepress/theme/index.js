import DefaultTheme from "vitepress/theme";
import VideoPlayer from "./VideoPlayer.vue";
import "./custom.css";

// Translation notice temporarily disabled — revert to default Layout
// To re-enable: import Layout from "./Layout.vue" and add Layout to the export below
export default {
  extends: DefaultTheme,
  enhanceApp({ app }) {
    app.component("VideoPlayer", VideoPlayer);
  },
};
