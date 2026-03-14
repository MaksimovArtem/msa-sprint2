import { ApolloServer } from '@apollo/server';
import { startStandaloneServer } from '@apollo/server/standalone';
import { buildSubgraphSchema } from '@apollo/subgraph';
import gql from 'graphql-tag';
import * as grpc from '@grpc/grpc-js';
import * as protoLoader from '@grpc/proto-loader';

const GRPC_TARGET = 'hotelio-monolith:9090';

const typeDefs = gql`
  extend type Hotel @key(fields: "id") {
    id: ID! @external
  }

  type Booking @key(fields: "id") {
    id: ID!
    userId: String!
    hotelId: String!
    promoCode: String
    discountPercent: Int

    hotel: Hotel
  }

  type Query {
    bookingsByUser(userId: String!): [Booking]
  }

`;

const packageDefinition = protoLoader.loadSync(
  new URL('./booking.proto', import.meta.url).pathname,
  {
    keepCase: false,
    longs: String,
    enums: String,
    defaults: true,
    oneofs: true,
  },
);

const proto = grpc.loadPackageDefinition(packageDefinition);
const bookingServiceClient = new proto.booking.BookingService(
  GRPC_TARGET,
  grpc.credentials.createInsecure(),
);

function getUserIdFromHeaders(req) {
  const value = req?.headers?.userid;
  if (Array.isArray(value)) return value[0];
  return value;
}

function listBookingsByUser(userId) {
  return new Promise((resolve, reject) => {
    bookingServiceClient.ListBookings({ userId }, (err, response) => {
      if (err) return reject(err);
      resolve(response?.bookings ?? []);
    });
  });
}

function mapGrpcBookingToGql(booking) {
  const discountPercentNumber =
    booking?.discountPercent == null ? null : Number(booking.discountPercent);

  return {
    id: String(booking?.id ?? ''),
    userId: String(booking?.userId ?? ''),
    hotelId: String(booking?.hotelId ?? ''),
    promoCode: booking?.promoCode ?? null,
    discountPercent:
      discountPercentNumber != null && Number.isFinite(discountPercentNumber)
        ? Math.round(discountPercentNumber)
        : null,
  };
}

const resolvers = {
  Query: {
    bookingsByUser: async (_, { userId }, { req }) => {
      const authUserId = getUserIdFromHeaders(req);
      if (!authUserId || authUserId !== userId) throw new Error("Request for invalid userId(check headers)");

      const bookings = await listBookingsByUser(userId);
      return bookings.map(mapGrpcBookingToGql);
    },
  },
  Booking: {
    hotel: (booking) => ({ __typename: 'Hotel', id: booking.hotelId }),
  },
};

const server = new ApolloServer({
  schema: buildSubgraphSchema([{ typeDefs, resolvers }]),
});

startStandaloneServer(server, {
  listen: { port: 4001 },
  context: async ({ req }) => ({ req }),
}).then(() => {
  console.log('✅ Booking subgraph ready at http://localhost:4001/');
});
