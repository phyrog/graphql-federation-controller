const express = require("express");
const app = express();
const axios = require("axios");

const backends = {
  accounts: "https://pw678w138q.sse.codesandbox.io/graphql",
  reviews: "https://0yo165yq9v.sse.codesandbox.io/graphql",
  products: "https://x7jn4y20pp.sse.codesandbox.io/graphql",
  inventory: "https://o5oxqmn7j9.sse.codesandbox.io/graphql"
};

async function getBackendSchema(backendName) {
  const res = await axios.post(backends[backendName], {
    operationName: "GetSchema",
    query: "{_service{sdl}}",
    variables: {}
  });
  console.log(await res.text());
  const j = await res.json();
  return j.data._service.sdl;
}

const serviceAnnotations = {
  "schema.graphql.org/name": "default",
  "schema.graphql.org/partial": "accounts",
  "schema.graphql.org/version": 1
};

app.get("/:graphid/storage-secret/:apikeyhash.json", (req, res) => {
  res.send('"test2"');
});
app.get(
  "/:secret/:graphvariant/v:federationversion/composition-config-link",
  (req, res) => {
    res.json({ configPath: "config/test2" });
  }
);
app.get("/config/:config", (req, res) => {
  res.json({
    formatVersion: 1,
    id: "test3",
    implementingServiceLocations: Object.keys(backends).map(n => ({
      name: n,
      path: `service/${n}`
    })),
    schemaHash: "foobar"
  });
});
app.get("/service/:service", (req, res) => {
  res.json({
    url: backends[req.params.service],
    partialSchemaPath: `schema/${req.params.service}`
  });
});
app.get("/schema/:service", async (req, res) => {
  const schema = await getBackendSchema(req.params.service);
  res.send(schema);
});

app.post("/api/ingress/traces", (req, res) => {
  res.end();
});

app.listen(3000, function() {
  console.log("Example app listening on port 3000!");
});
