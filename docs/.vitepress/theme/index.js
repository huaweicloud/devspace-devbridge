import DefaultTheme from "vitepress/theme";
import VideoPlayer from "./VideoPlayer.vue";
import Layout from "./Layout.vue";
import "./custom.css";

export default {
  extends: DefaultTheme,
  Layout,
  enhanceApp({ app }) {
    app.component("VideoPlayer", VideoPlayer);
  },
};
