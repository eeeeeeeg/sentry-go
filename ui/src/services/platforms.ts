export type PlatformCategoryId = "popular" | "browser" | "server" | "mobile" | "desktop" | "all";

export type PlatformOption = {
  id: string;
  name: string;
  category: Exclude<PlatformCategoryId, "popular" | "all"> | "other";
};

export type PlatformCategory = {
  id: PlatformCategoryId;
  name: string;
  options: PlatformOption[];
};

const platformOptions: PlatformOption[] = [
  { id: "android", name: "Android", category: "mobile" },
  { id: "apple", name: "Apple", category: "mobile" },
  { id: "apple-ios", name: "iOS", category: "mobile" },
  { id: "apple-macos", name: "macOS", category: "desktop" },
  { id: "bun", name: "Bun", category: "server" },
  { id: "capacitor", name: "Capacitor", category: "mobile" },
  { id: "clojure", name: "Clojure", category: "server" },
  { id: "cordova", name: "Cordova", category: "mobile" },
  { id: "dart", name: "Dart", category: "mobile" },
  { id: "deno", name: "Deno", category: "server" },
  { id: "dotnet", name: ".NET", category: "server" },
  { id: "dotnet-aspnet", name: "ASP.NET", category: "server" },
  { id: "dotnet-aspnetcore", name: "ASP.NET Core", category: "server" },
  { id: "dotnet-awslambda", name: ".NET AWS Lambda", category: "server" },
  { id: "dotnet-gcpfunctions", name: ".NET GCP Functions", category: "server" },
  { id: "dotnet-maui", name: ".NET MAUI", category: "mobile" },
  { id: "dotnet-uwp", name: ".NET UWP", category: "desktop" },
  { id: "dotnet-winforms", name: ".NET WinForms", category: "desktop" },
  { id: "dotnet-wpf", name: ".NET WPF", category: "desktop" },
  { id: "dotnet-xamarin", name: "Xamarin", category: "mobile" },
  { id: "electron", name: "Electron", category: "desktop" },
  { id: "elixir", name: "Elixir", category: "server" },
  { id: "flutter", name: "Flutter", category: "mobile" },
  { id: "go", name: "Go", category: "server" },
  { id: "go-echo", name: "Echo", category: "server" },
  { id: "go-fasthttp", name: "FastHTTP", category: "server" },
  { id: "go-fiber", name: "Fiber", category: "server" },
  { id: "go-gin", name: "Gin", category: "server" },
  { id: "go-http", name: "Go HTTP", category: "server" },
  { id: "go-iris", name: "Iris", category: "server" },
  { id: "go-martini", name: "Martini", category: "server" },
  { id: "go-negroni", name: "Negroni", category: "server" },
  { id: "godot", name: "Godot", category: "desktop" },
  { id: "ionic", name: "Ionic", category: "mobile" },
  { id: "java", name: "Java", category: "server" },
  { id: "java-log4j2", name: "Log4j 2", category: "server" },
  { id: "java-logback", name: "Logback", category: "server" },
  { id: "java-spring", name: "Spring", category: "server" },
  { id: "java-spring-boot", name: "Spring Boot", category: "server" },
  { id: "javascript", name: "JavaScript", category: "browser" },
  { id: "javascript-angular", name: "Angular", category: "browser" },
  { id: "javascript-astro", name: "Astro", category: "browser" },
  { id: "javascript-ember", name: "Ember", category: "browser" },
  { id: "javascript-gatsby", name: "Gatsby", category: "browser" },
  { id: "javascript-nextjs", name: "Next.js", category: "browser" },
  { id: "javascript-nuxt", name: "Nuxt", category: "browser" },
  { id: "javascript-react", name: "React", category: "browser" },
  { id: "javascript-react-router", name: "React Router", category: "browser" },
  { id: "javascript-remix", name: "Remix", category: "browser" },
  { id: "javascript-solid", name: "Solid", category: "browser" },
  { id: "javascript-solidstart", name: "SolidStart", category: "browser" },
  { id: "javascript-svelte", name: "Svelte", category: "browser" },
  { id: "javascript-sveltekit", name: "SvelteKit", category: "browser" },
  { id: "javascript-tanstackstart-react", name: "TanStack Start", category: "browser" },
  { id: "javascript-vue", name: "Vue", category: "browser" },
  { id: "kotlin", name: "Kotlin", category: "mobile" },
  { id: "minidump", name: "Minidump", category: "desktop" },
  { id: "native", name: "Native", category: "desktop" },
  { id: "native-qt", name: "Qt", category: "desktop" },
  { id: "nintendo-switch", name: "Nintendo Switch", category: "other" },
  { id: "node", name: "Node.js", category: "server" },
  { id: "node-awslambda", name: "Node AWS Lambda", category: "server" },
  { id: "node-azurefunctions", name: "Node Azure Functions", category: "server" },
  { id: "node-cloudflare-pages", name: "Cloudflare Pages", category: "server" },
  { id: "node-cloudflare-workers", name: "Cloudflare Workers", category: "server" },
  { id: "node-connect", name: "Connect", category: "server" },
  { id: "node-express", name: "Express", category: "server" },
  { id: "node-fastify", name: "Fastify", category: "server" },
  { id: "node-gcpfunctions", name: "Node GCP Functions", category: "server" },
  { id: "node-hapi", name: "Hapi", category: "server" },
  { id: "node-hono", name: "Hono", category: "server" },
  { id: "node-koa", name: "Koa", category: "server" },
  { id: "node-nestjs", name: "NestJS", category: "server" },
  { id: "other", name: "Other", category: "other" },
  { id: "perl", name: "Perl", category: "server" },
  { id: "php", name: "PHP", category: "server" },
  { id: "php-laravel", name: "Laravel", category: "server" },
  { id: "php-symfony", name: "Symfony", category: "server" },
  { id: "playstation", name: "PlayStation", category: "other" },
  { id: "powershell", name: "PowerShell", category: "server" },
  { id: "python", name: "Python", category: "server" },
  { id: "python-aiohttp", name: "AIOHTTP", category: "server" },
  { id: "python-asgi", name: "ASGI", category: "server" },
  { id: "python-awslambda", name: "Python AWS Lambda", category: "server" },
  { id: "python-bottle", name: "Bottle", category: "server" },
  { id: "python-celery", name: "Celery", category: "server" },
  { id: "python-chalice", name: "Chalice", category: "server" },
  { id: "python-django", name: "Django", category: "server" },
  { id: "python-falcon", name: "Falcon", category: "server" },
  { id: "python-fastapi", name: "FastAPI", category: "server" },
  { id: "python-flask", name: "Flask", category: "server" },
  { id: "python-gcpfunctions", name: "Python GCP Functions", category: "server" },
  { id: "python-litestar", name: "Litestar", category: "server" },
  { id: "python-pylons", name: "Pylons", category: "server" },
  { id: "python-pymongo", name: "PyMongo", category: "server" },
  { id: "python-pyramid", name: "Pyramid", category: "server" },
  { id: "python-quart", name: "Quart", category: "server" },
  { id: "python-rq", name: "RQ", category: "server" },
  { id: "python-sanic", name: "Sanic", category: "server" },
  { id: "python-serverless", name: "Python Serverless", category: "server" },
  { id: "python-starlette", name: "Starlette", category: "server" },
  { id: "python-tornado", name: "Tornado", category: "server" },
  { id: "python-tryton", name: "Tryton", category: "server" },
  { id: "python-wsgi", name: "WSGI", category: "server" },
  { id: "react-native", name: "React Native", category: "mobile" },
  { id: "ruby", name: "Ruby", category: "server" },
  { id: "ruby-rack", name: "Rack", category: "server" },
  { id: "ruby-rails", name: "Rails", category: "server" },
  { id: "rust", name: "Rust", category: "server" },
  { id: "unity", name: "Unity", category: "desktop" },
  { id: "unreal", name: "Unreal Engine", category: "desktop" },
  { id: "xbox", name: "Xbox", category: "other" },
];

const popularPlatformIds = [
  "javascript",
  "javascript-react",
  "javascript-vue",
  "javascript-angular",
  "javascript-nextjs",
  "node",
  "python",
  "python-django",
  "python-flask",
  "java-spring-boot",
  "go",
  "php-laravel",
  "ruby-rails",
  "android",
  "apple-ios",
  "flutter",
  "react-native",
];

const platformMap = new Map(platformOptions.map((option) => [option.id, option]));

export const platformCategories: PlatformCategory[] = [
  { id: "popular", name: "Popular", options: popularPlatformIds.map((id) => platformMap.get(id)).filter(Boolean) as PlatformOption[] },
  { id: "browser", name: "Browser", options: platformOptions.filter((option) => option.category === "browser") },
  { id: "server", name: "Server", options: platformOptions.filter((option) => option.category === "server") },
  { id: "mobile", name: "Mobile", options: platformOptions.filter((option) => option.category === "mobile") },
  { id: "desktop", name: "Desktop", options: platformOptions.filter((option) => option.category === "desktop") },
  { id: "all", name: "All", options: [...platformOptions].sort((a, b) => a.name.localeCompare(b.name)) },
];

export function getPlatformLabel(platform: string) {
  return platformMap.get(platform)?.name ?? platform;
}

export function getPlatformCategory(platform: string): PlatformCategoryId {
  if (popularPlatformIds.includes(platform)) {
    return "popular";
  }
  const category = platformMap.get(platform)?.category;
  if (category === "browser" || category === "server" || category === "mobile" || category === "desktop") {
    return category;
  }
  return "all";
}
