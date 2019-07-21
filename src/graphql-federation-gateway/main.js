const fs = require("fs");
const minimist = require("minimist");
const { ApolloGateway } = require("@apollo/gateway");
const { ApolloServer } = require("apollo-server");

const argv = minimist(process.argv.slice(2));

const configFile = argv.config || "/endpoint-config/config.json";

let gateway;
let server;

gateway = new ApolloGateway();

/*
storageSecretUrl = "<secret-base>/<graph_id>/storage-secret/<api-key-hash>.json"; => quoted string?
linkFile = "<partial-base>/<secret>/<graph-variant>/v<federation-version>/composition-config-link"; => {configPath: string}
configFileResultPath = "<partial-base>/<configPath>"; => {result: ConfigFileResult}
serviceLocation = "<partial-base>/<parsedConfig.implementingServiceLocations.<key>>"; => {url: string, partialSchemaPath: string}
schemaPath = <partial-base>/<partialSchemaPath>; => graphql schema
*/

server = new ApolloServer({
  gateway: gateway,
  subscriptions: false,
  engine: {
    apiKey: "secret:api:key",
    graphId: "graphId",
    apiKeyHash: "apiKeyHash",
    graphVariant: "graphVariant",
    endpointUrl: "http://localhost:8000"
  }
});
server.listen().then(url => {
  console.log(`Listening to ${url}.`);
});
