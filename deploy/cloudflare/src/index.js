import { Container, getContainer } from "@cloudflare/containers";
import { env as workerEnv } from "cloudflare:workers";

const DEFAULT_INSTANCE_NAME = "default";
const INSTANCE_PARAM = "instance";

function sanitizeInstanceName(value) {
  if (!value) {
    return DEFAULT_INSTANCE_NAME;
  }
  const normalized = value.trim().toLowerCase().replace(/[^a-z0-9_-]+/g, "-");
  const compact = normalized.replace(/-+/g, "-").slice(0, 63).replace(/^-|-$/g, "");
  return compact || DEFAULT_INSTANCE_NAME;
}

function stringOrDefault(value, fallback) {
  if (typeof value !== "string") {
    return fallback;
  }
  const trimmed = value.trim();
  return trimmed || fallback;
}

function optionalSecret(value) {
  if (typeof value !== "string") {
    return null;
  }
  const trimmed = value.trim();
  return trimmed || null;
}

export class MisterMorphContainer extends Container {
  defaultPort = 8787;
  sleepAfter = "15m";

  get envVars() {
    const vars = {
      MISTER_MORPH_LLM_PROVIDER: stringOrDefault(workerEnv.MISTER_MORPH_LLM_PROVIDER, "openai"),
      MISTER_MORPH_LLM_ENDPOINT: stringOrDefault(workerEnv.MISTER_MORPH_LLM_ENDPOINT, "https://api.openai.com"),
      MISTER_MORPH_LLM_MODEL: stringOrDefault(workerEnv.MISTER_MORPH_LLM_MODEL, "gpt-5.2"),
      MISTER_MORPH_LOG_LEVEL: stringOrDefault(workerEnv.MISTER_MORPH_LOG_LEVEL, "info"),
      MISTER_MORPH_LOG_FORMAT: "json",
      MISTER_MORPH_TOOLS_BASH_ENABLED: stringOrDefault(workerEnv.MISTER_MORPH_TOOLS_BASH_ENABLED, "false"),
      MISTER_MORPH_FILE_STATE_DIR: "/data/state",
      MISTER_MORPH_FILE_CACHE_DIR: "/data/cache",
    };

    const llmAPIKey = optionalSecret(workerEnv.MISTER_MORPH_LLM_API_KEY);
    if (llmAPIKey) {
      vars.MISTER_MORPH_LLM_API_KEY = llmAPIKey;
    }

    const serverAuthToken = optionalSecret(workerEnv.MISTER_MORPH_SERVER_AUTH_TOKEN);
    if (serverAuthToken) {
      vars.MISTER_MORPH_SERVER_AUTH_TOKEN = serverAuthToken;
    }

    const telegramToken = optionalSecret(workerEnv.MISTER_MORPH_TELEGRAM_BOT_TOKEN);
    if (telegramToken) {
      vars.MISTER_MORPH_TELEGRAM_BOT_TOKEN = telegramToken;
    }

    return vars;
  }
}

export default {
  async fetch(request, env) {
    const url = new URL(request.url);
    const instanceName = sanitizeInstanceName(url.searchParams.get(INSTANCE_PARAM));

    url.searchParams.delete(INSTANCE_PARAM);
    const upstreamRequest = new Request(url.toString(), request);

    return getContainer(env.MISTER_MORPH_CONTAINER, instanceName).fetch(upstreamRequest);
  },
};
