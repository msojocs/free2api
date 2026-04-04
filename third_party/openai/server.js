// create http server for openai token
const http = require("http");
const url = require("url");

const server = http.createServer(async (req, res) => {
  const parsedUrl = url.parse(req.url, true);
  if (parsedUrl.pathname === "/proof") {
    try {
      const { TokenGenerator } = require("./utils/proof");
      const generator = new TokenGenerator();
      const proof = await generator.getRequirementsToken();
      res.writeHead(200, { "Content-Type": "application/json" });
      res.end(JSON.stringify({ token: proof }));
    } catch (error) {
      res.writeHead(500, { "Content-Type": "application/json" });
      res.end(JSON.stringify({ error: error.message }));
    }
  } else if (parsedUrl.pathname === "/turnstile") {
    try {
      const body = req.method === "POST" ? await new Promise((resolve) => {
        let data = "";
        req.on("data", (chunk) => {
          data += chunk;
        });
        req.on("end", () => {
          resolve(data);
        });
      }) : null;
      const data = JSON.parse(body);
      const { turnstile } = require("./utils/turnstile");
      const result = await turnstile(data.proof, data.sentinelInfo);
      res.writeHead(200, { "Content-Type": "application/json" });
      res.end(JSON.stringify({ turnstile: result }));
    } catch (error) {
      res.writeHead(500, { "Content-Type": "application/json" });
      res.end(JSON.stringify({ error: error.message }));
    }
  } else {
    res.writeHead(404, { "Content-Type": "application/json" });
    res.end(JSON.stringify({ error: "Not found" }));
  }
});

const PORT = process.env.PORT || 3000;
server.listen(PORT, () => {
  console.log(`OpenAI Token Server is running on port ${PORT}`);
});
