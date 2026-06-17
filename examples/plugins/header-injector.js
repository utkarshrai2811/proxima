export const meta = {
  name: "Header Injector",
  version: "1.0.0",
  description: "Adds an X-Proxima-Plugin header to every proxied request",
  author: "Proxima User"
};

export function onRequest(ctx) {
  ctx.request.headers['X-Proxima-Plugin'] = 'header-injector/1.0';
  return ctx;
}
