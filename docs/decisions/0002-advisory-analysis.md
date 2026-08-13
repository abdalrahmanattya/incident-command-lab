# ADR 0002: analysis never remediates

Status: accepted — 2026-08-13

Incident analysis returns cited hypotheses and checks only. Ollama and Azure
OpenAI are interchangeable advisory adapters selected by environment. Remote
analysis is disabled unless explicitly enabled; deterministic fallback remains
available when a provider fails.
