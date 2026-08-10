import { useState } from "react";
import { Check } from "@phosphor-icons/react";
import { Reveal } from "./shared/Reveal";
import { Section } from "./shared/Section";
import { useI18n } from "../../i18n/I18nProvider";
import type { MessageKey } from "../../i18n/messages";

/**
 * UseCases: accessible tabs. Keyboard-driven (roving tabindex + arrow keys
 * would be ideal; buttons with aria-selected + role=tablist are the base).
 */
const USE_CASES: Array<{
  id: string;
  labelKey: MessageKey;
  titleKey: MessageKey;
  bodyKey: MessageKey;
  pointKeys: MessageKey[];
}> = [
  {
    id: "reliability",
    labelKey: "landing.useCases.reliability.label",
    titleKey: "landing.useCases.reliability.title",
    bodyKey: "landing.useCases.reliability.body",
    pointKeys: [
      "landing.useCases.reliability.point1",
      "landing.useCases.reliability.point2",
      "landing.useCases.reliability.point3",
    ],
  },
  {
    id: "operations",
    labelKey: "landing.useCases.operations.label",
    titleKey: "landing.useCases.operations.title",
    bodyKey: "landing.useCases.operations.body",
    pointKeys: [
      "landing.useCases.operations.point1",
      "landing.useCases.operations.point2",
      "landing.useCases.operations.point3",
    ],
  },
  {
    id: "enterprise",
    labelKey: "landing.useCases.enterprise.label",
    titleKey: "landing.useCases.enterprise.title",
    bodyKey: "landing.useCases.enterprise.body",
    pointKeys: [
      "landing.useCases.enterprise.point1",
      "landing.useCases.enterprise.point2",
      "landing.useCases.enterprise.point3",
    ],
  },
];

export function UseCases() {
  const { t } = useI18n();
  const [active, setActive] = useState(0);
  const current = USE_CASES[active];

  return (
    <Section
      id="use-cases"
      eyebrow={t("landing.useCases.eyebrow")}
      title={t("landing.useCases.title")}
    >
      <div style={{ marginTop: 40 }}>
        <div
          role="tablist"
          aria-label={t("landing.useCases.tablistAria")}
          style={{
            display: "flex",
            gap: 8,
            flexWrap: "wrap",
            borderBottom: "1px solid var(--line)",
            paddingBottom: 0,
          }}
        >
          {USE_CASES.map((useCase, index) => (
            <button
              key={useCase.id}
              role="tab"
              id={`use-case-tab-${useCase.id}`}
              aria-selected={active === index}
              aria-controls={`use-case-panel-${useCase.id}`}
              onClick={() => setActive(index)}
              style={{
                appearance: "none",
                border: "none",
                borderBottom:
                  active === index
                    ? "2px solid var(--brand)"
                    : "2px solid transparent",
                background: "transparent",
                color: active === index ? "var(--ink)" : "var(--ink-muted)",
                padding: "12px 16px",
                fontSize: 14,
                fontWeight: 600,
                cursor: "pointer",
                marginBottom: -1,
              }}
            >
              {t(useCase.labelKey)}
            </button>
          ))}
        </div>
        <Reveal key={current.id}>
          <div
            role="tabpanel"
            id={`use-case-panel-${current.id}`}
            aria-labelledby={`use-case-tab-${current.id}`}
            style={{
              paddingTop: 36,
              display: "grid",
              gridTemplateColumns: "1fr",
              gap: 24,
            }}
          >
            <div>
              <h3 style={{ fontSize: 24, fontWeight: 700, color: "var(--ink)", margin: 0 }}>
                {t(current.titleKey)}
              </h3>
              <p
                style={{
                  marginTop: 14,
                  fontSize: 15.5,
                  lineHeight: 1.65,
                  color: "var(--ink-soft)",
                  maxWidth: "58ch",
                }}
              >
                {t(current.bodyKey)}
              </p>
            </div>
            <ul
              style={{
                listStyle: "none",
                margin: 0,
                padding: 0,
                display: "grid",
                gap: 12,
              }}
            >
              {current.pointKeys.map((pointKey) => (
                <li
                  key={pointKey}
                  style={{
                    display: "flex",
                    gap: 12,
                    alignItems: "flex-start",
                    fontSize: 14.5,
                    color: "var(--ink)",
                    lineHeight: 1.5,
                  }}
                >
                  <span
                    aria-hidden="true"
                    style={{
                      width: 20,
                      height: 20,
                      borderRadius: "50%",
                      background: "color-mix(in srgb, var(--brand) 14%, transparent)",
                      color: "var(--brand)",
                      display: "grid",
                      placeItems: "center",
                      flexShrink: 0,
                      marginTop: 1,
                    }}
                  >
                    <Check size={12} weight="bold" />
                  </span>
                  {t(pointKey)}
                </li>
              ))}
            </ul>
          </div>
        </Reveal>
      </div>
    </Section>
  );
}
