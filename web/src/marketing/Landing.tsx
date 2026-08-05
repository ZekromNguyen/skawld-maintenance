import { SiteHeader } from "./components/SiteHeader";
import { Hero } from "./components/Hero";
import { LogoWall } from "./components/LogoWall";
import { FeaturesBento } from "./components/FeaturesBento";
import { WorkflowRail } from "./components/WorkflowRail";
import { Integrations } from "./components/Integrations";
import { SecurityReliability } from "./components/SecurityReliability";
import { UseCases } from "./components/UseCases";
import { PilotCta } from "./components/PilotCta";
import { Testimonials } from "./components/Testimonials";
import { Faq } from "./components/Faq";
import { SiteFooter } from "./components/SiteFooter";

/**
 * Landing: public marketing page for Skawld (route: /landing).
 * Built against docs/contributing/web-ui-design-system.md.
 */
export function Landing() {
  return (
    <div className="landing">
      <a
        href="#main-content"
        style={{
          position: "absolute",
          left: -9999,
          top: 0,
          zIndex: 100,
          background: "var(--raised)",
          color: "var(--ink)",
          padding: "10px 16px",
          borderRadius: 8,
        }}
        onFocus={(e) => {
          e.currentTarget.style.left = "16px";
          e.currentTarget.style.top = "16px";
        }}
        onBlur={(e) => {
          e.currentTarget.style.left = "-9999px";
        }}
      >
        Skip to content
      </a>
      <SiteHeader />
      <main id="main-content">
        <Hero />
        <LogoWall />
        <FeaturesBento />
        <WorkflowRail />
        <Integrations />
        <SecurityReliability />
        <UseCases />
        <Testimonials />
        <PilotCta />
        <Faq />
      </main>
      <SiteFooter />
    </div>
  );
}
