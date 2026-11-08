// Client-side routes such as /groups/123 have no object in S3. Serve the app
// shell for any path whose last segment has no file extension. A custom error
// response would do the same but would also swallow the API's 403s and 404s.
function handler(event) {
  var request = event.request
  if (!/\.[A-Za-z0-9]+$/.test(request.uri)) {
    request.uri = '/index.html'
  }
  return request
}
