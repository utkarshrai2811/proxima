export const meta = {
  name: "5xx Logger",
  version: "1.0.0",
  description: "Logs all 5xx responses via proxima.log()",
  author: "Proxima User"
};

export function onResponse(ctx) {
  if (ctx.response && ctx.response.statusCode >= 500) {
    proxima.log('5xx: ' + ctx.response.statusCode + ' — ' + (ctx.request ? ctx.request.url : ''));
  }
  return ctx;
}
