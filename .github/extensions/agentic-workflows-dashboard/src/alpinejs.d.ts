declare module "alpinejs" {
  interface AlpineStatic {
    data<T extends object>(name: string, factory: () => T): void;
    start(): void;
  }
  const Alpine: AlpineStatic;
  export default Alpine;
}
