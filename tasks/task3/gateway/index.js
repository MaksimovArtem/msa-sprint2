import { ApolloServer } from '@apollo/server';
import { startStandaloneServer } from '@apollo/server/standalone';
import { ApolloGateway, RemoteGraphQLDataSource } from '@apollo/gateway';
import { HotelCachingDataSource } from './hotelCacheDataSource.js';

function getUserIdFromHeaders(req) {
  const value = req?.headers?.userid;
  if (Array.isArray(value)) return value[0];
  return value;
}

const gateway = new ApolloGateway({
  buildService: ({ name, url }) =>
    new (name === 'hotel' ? HotelCachingDataSource : RemoteGraphQLDataSource)({
      url,
      willSendRequest({ request, context }) {
        const userId = getUserIdFromHeaders(context?.req);
        if (userId) request.http.headers.set('userid', String(userId));
      },
    }),
  serviceList: [
    { name: 'booking', url: 'http://booking-subgraph:4001' },
    { name: 'hotel',   url: 'http://hotel-subgraph:4002' },
    { name: 'promo',   url: 'http://promo-subgraph:4003' },
  ]
});

const server = new ApolloServer({ gateway, subscriptions: false });

startStandaloneServer(server, {
  listen: { port: 4000 },
  context: async ({ req }) => ({ req, hotelRequestCache: new Map() }),
}).then(({ url }) => {
  console.log(`🚀 Gateway ready at ${url}`);
});
