import { ApolloServer } from '@apollo/server';
import { startStandaloneServer } from '@apollo/server/standalone';
import { buildSubgraphSchema } from '@apollo/subgraph';
import gql from 'graphql-tag';

const HOTEL_API_BASE_URL = 'http://hotelio-monolith:8084';

const typeDefs = gql`
  type Hotel @key(fields: "id") {
    id: ID!
    name: String
    city: String
    stars: Float
  }

  type Query {
    hotelsByIds(ids: [ID!]!): [Hotel]
  }
`;

function hotelByIdUrl(id) {
  return `${HOTEL_API_BASE_URL}/api/hotels/${encodeURIComponent(String(id))}`;
}

async function fetchHotelsByIds(ids) {
  if (!Array.isArray(ids) || ids.length === 0) return [];

  const uniqueIds = [...new Set(ids.map((value) => String(value)))];
  const pairs = await Promise.all(
    uniqueIds.map(async (uniqueId) => [uniqueId, await fetchHotelById(uniqueId)]),
  );

  const byId = new Map(pairs);
  return ids.map((value) => byId.get(String(value)) ?? null);
}

async function fetchHotelById(id) {
  const abortController = new AbortController();
  const timeout = setTimeout(() => abortController.abort(), 2000);

  try {
    const response = await fetch(hotelByIdUrl(id), { signal: abortController.signal });
    if (response.status === 404) return null;
    if (!response.ok) {
      throw new Error(`Hotel API error: ${response.status} ${response.statusText}`);
    }

    const hotel = await response.json();
    if (!hotel) return null;

    return {
      id: String(hotel.id ?? id),
      name: hotel.name ?? String(hotel.id ?? id),
      city: hotel.city ?? null,
      stars: hotel.rating ?? null,
    };
  } finally {
    clearTimeout(timeout);
  }
}

const resolvers = {
  Hotel: {
    __resolveReference: async ({ id }) => {
      return fetchHotelById(id);
    },
  },
  Query: {
    hotelsByIds: async (_, { ids }) => {
      return fetchHotelsByIds(ids);
    },
  },
};

const server = new ApolloServer({
  schema: buildSubgraphSchema([{ typeDefs, resolvers }]),
});

startStandaloneServer(server, {
  listen: { port: 4002 },
}).then(() => {
  console.log('✅ Hotel subgraph ready at http://localhost:4002/');
});
