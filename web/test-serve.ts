import { serve } from "bun";

const server = serve({
  hostname: "0.0.0.0",
  port: 3000,
  routes: {
    "/": new Response("Welcome to Bun!"),
    "/api/validate": async (req) => {
      console.log({ req });
      try {
        return Response.json({ message: "OK" }, { status: 400 }); // 100 - 599
      } catch (error) {
        return Response.json({ error: error }, { status: 500 });
      }
    },
  },
});

console.log(`Listening on localhost:${server.port}`);
