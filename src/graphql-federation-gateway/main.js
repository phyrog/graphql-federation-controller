const { ApolloGateway } = require("@apollo/gateway");
const { ApolloServer } = require("apollo-server");

const gateway = new ApolloGateway();
const server = new ApolloServer({
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
server.listen().then(({url}) => {
  console.log(`Listening to ${url}.`);
});
