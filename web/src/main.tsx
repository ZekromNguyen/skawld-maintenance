import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { App } from "./App";
import { Landing } from "./marketing/Landing";
import { I18nProvider } from "./i18n/I18nProvider";
import { ThemeProvider } from "./theme/ThemeProvider";
import "@fontsource-variable/geist";
import "@fontsource-variable/geist-mono";
import "./styles.css";
import "./marketing/marketing.css";

const root = createRoot(document.getElementById("root")!);

// Public marketing page at /landing; the operator console stays at root.
if (window.location.pathname === "/landing") {
  root.render(
    <StrictMode>
      <ThemeProvider>
        <I18nProvider>
          <Landing />
        </I18nProvider>
      </ThemeProvider>
    </StrictMode>,
  );
} else {
  root.render(
    <StrictMode>
      <ThemeProvider>
        <I18nProvider>
          <App />
        </I18nProvider>
      </ThemeProvider>
    </StrictMode>,
  );
}
