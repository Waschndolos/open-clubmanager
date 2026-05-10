from __future__ import annotations

import json
from pathlib import Path
from textwrap import dedent

import yaml

ROOT = Path(__file__).resolve().parent.parent
OPENAPI_PATH = ROOT / 'openapi.yaml'
OUTPUT_PATH = ROOT / 'docs' / 'index.html'
DOCS_OPENAPI_PATH = ROOT / 'docs' / 'openapi.yaml'
NOJEKYLL_PATH = ROOT / 'docs' / '.nojekyll'


def build_html(spec: dict) -> str:
    spec_json = json.dumps(spec, ensure_ascii=False, separators=(',', ':'))

    return dedent(
        f"""\
        <!doctype html>
        <html lang="de">
          <head>
            <meta charset="utf-8" />
            <meta name="viewport" content="width=device-width, initial-scale=1" />
            <title>Open ClubManager API Docs</title>
            <meta
              name="description"
              content="Interaktive ReDoc-Dokumentation für die Open ClubManager REST API."
            />
            <style>
              :root {{
                color-scheme: light dark;
              }}

              body {{
                margin: 0;
                font-family: Inter, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
                background: #f5f7fb;
                color: #1f2937;
              }}

              .topbar {{
                position: sticky;
                top: 0;
                z-index: 10;
                display: flex;
                flex-wrap: wrap;
                gap: 0.75rem;
                align-items: center;
                justify-content: space-between;
                padding: 0.9rem 1.25rem;
                background: rgba(17, 24, 39, 0.96);
                color: #fff;
                box-shadow: 0 2px 10px rgba(0, 0, 0, 0.18);
              }}

              .topbar__title {{
                font-size: 1rem;
                font-weight: 700;
                letter-spacing: 0.01em;
              }}

              .topbar__links {{
                display: flex;
                flex-wrap: wrap;
                gap: 0.75rem;
              }}

              .topbar__links a {{
                color: #93c5fd;
                text-decoration: none;
                font-weight: 600;
              }}

              .topbar__links a:hover {{
                text-decoration: underline;
              }}

              #redoc-container {{
                min-height: calc(100vh - 64px);
              }}
            </style>
          </head>
          <body>
            <header class="topbar">
              <div class="topbar__title">Open ClubManager – API Dokumentation</div>
              <nav class="topbar__links" aria-label="Dokumentationslinks">
                <a href="./openapi.yaml">OpenAPI YAML</a>
              </nav>
            </header>

            <div id="redoc-container"></div>

            <script>
              window.__OPENAPI_SPEC__ = {spec_json};
            </script>
            <script src="https://cdn.redoc.ly/redoc/latest/bundles/redoc.standalone.js"></script>
            <script>
              Redoc.init(window.__OPENAPI_SPEC__, {{
                hideDownloadButton: false,
                expandResponses: '200,201',
                pathInMiddlePanel: true,
                requiredPropsFirst: true,
                sortPropsAlphabetically: false,
                theme: {{
                  colors: {{
                    primary: {{ main: '#2563eb' }}
                  }},
                  typography: {{
                    fontSize: '15px',
                    fontFamily: 'Inter, system-ui, sans-serif',
                    headings: {{ fontFamily: 'Inter, system-ui, sans-serif' }}
                  }}
                }}
              }}, document.getElementById('redoc-container'));
            </script>
          </body>
        </html>
        """
    )


def main() -> None:
    spec = yaml.safe_load(OPENAPI_PATH.read_text(encoding='utf-8'))
    OUTPUT_PATH.parent.mkdir(parents=True, exist_ok=True)
    OUTPUT_PATH.write_text(build_html(spec), encoding='utf-8')
    DOCS_OPENAPI_PATH.write_text(OPENAPI_PATH.read_text(encoding='utf-8'), encoding='utf-8')
    NOJEKYLL_PATH.write_text('', encoding='utf-8')
    print(f'Generated {OUTPUT_PATH.relative_to(ROOT)} from {OPENAPI_PATH.relative_to(ROOT)}')


if __name__ == '__main__':
    main()

