#!/bin/sh
poetry run uvicorn app.api.main:app --host 0.0.0.0 --port "$PORT"