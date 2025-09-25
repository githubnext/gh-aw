interface SafeOutputConfig {
  type: string;
  max?: number;
}

type SafeOutputConfigs = Record<string, SafeOutputConfig>;

export { SafeOutputConfig, SafeOutputConfigs };
