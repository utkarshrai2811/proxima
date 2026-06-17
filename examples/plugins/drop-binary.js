export const meta = {
  name: "Drop Binary Responses",
  version: "1.0.0",
  description: "Auto-drops image/video/audio content from the intercept queue",
  author: "Proxima User"
};

export function onIntercept(ctx) {
  const ct = (ctx.response && ctx.response.headers['Content-Type']) || '';
  if (ct.startsWith('image/') || ct.startsWith('video/') || ct.startsWith('audio/')) {
    ctx.action = 'drop';
  }
  return ctx;
}
