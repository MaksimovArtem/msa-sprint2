import { ApolloServer } from '@apollo/server';
import { startStandaloneServer } from '@apollo/server/standalone';
import { buildSubgraphSchema } from '@apollo/subgraph';
import gql from 'graphql-tag';

const PROMO_API_BASE_URL = 'http://hotelio-monolith:8084';

const typeDefs = gql`
  extend type Booking @key(fields: "id") {
    id: ID! @external
    promoCode: String @external
    discountPercent: Int @external
    discountInfo(userId: ID): DiscountInfo! @requires(fields: "promoCode")
  }
  
  type DiscountInfo {
    isValid: Boolean!
    originalDiscount: Float!    # Исходное значение из booking
    finalDiscount: Float!       # Актуальное значение после проверки
    description: String
    expiresAt: String
  }
  
  type Query {
    validatePromoCode(code: String!, userId: ID): DiscountInfo!
  }
`;

function getUserIdFromHeaders(req) {
  const value = req?.headers?.userid;
  if (Array.isArray(value)) return value[0];
  return value;
}

function promoByCodeUrl(code) {
  return `${PROMO_API_BASE_URL}/api/promos/${encodeURIComponent(String(code))}`;
}

function validatePromoUrl(code, userId) {
  const url = new URL(`${PROMO_API_BASE_URL}/api/promos/validate`);
  url.searchParams.set('code', String(code));
  url.searchParams.set('userId', String(userId));
  return url.toString();
}

async function fetchPromoByCode(code) {
  const abortController = new AbortController();
  const timeout = setTimeout(() => abortController.abort(), 2000);

  try {
    const response = await fetch(promoByCodeUrl(code), { signal: abortController.signal });
    if (response.status === 404) return null;
    if (!response.ok) {
      throw new Error(`Promo API error: ${response.status} ${response.statusText}`);
    }
    return response.json();
  } finally {
    clearTimeout(timeout);
  }
}

async function validatePromoCodeViaApi(code, userId) {
  const abortController = new AbortController();
  const timeout = setTimeout(() => abortController.abort(), 2000);

  try {
    const url = validatePromoUrl(code, userId);

    const postResponse = await fetch(url, { method: 'POST', signal: abortController.signal });
    if (postResponse.ok) return true;
    if (postResponse.status === 400 || postResponse.status === 404) return false;

    if (postResponse.status === 405) {
      const getResponse = await fetch(url, { method: 'GET', signal: abortController.signal });
      if (getResponse.ok) return true;
      if (getResponse.status === 400 || getResponse.status === 404) return false;
      throw new Error(`Promo validate error: ${getResponse.status} ${getResponse.statusText}`);
    }

    throw new Error(`Promo validate error: ${postResponse.status} ${postResponse.statusText}`);
  } finally {
    clearTimeout(timeout);
  }
}

const resolvers = {
  Booking: {
    discountInfo: async (booking, { userId }, { req }) => {
      const code = booking?.promoCode;
      const authUserId = userId ?? getUserIdFromHeaders(req);

      const originalDiscountNumber =
        booking?.discountPercent == null ? 0 : Number(booking.discountPercent);
      const originalDiscount = Number.isFinite(originalDiscountNumber) ? originalDiscountNumber : 0;

      if (!code || !authUserId) {
        return {
          isValid: false,
          originalDiscount,
          finalDiscount: 0,
          description: null,
          expiresAt: null,
        };
      }

      const [isValid, promo] = await Promise.all([
        validatePromoCodeViaApi(code, authUserId),
        fetchPromoByCode(code),
      ]);

      const promoDiscountNumber = promo?.discount == null ? 0 : Number(promo.discount);
      const promoDiscount = Number.isFinite(promoDiscountNumber) ? promoDiscountNumber : 0;

      return {
        isValid,
        originalDiscount,
        finalDiscount: isValid ? promoDiscount : 0,
        description: promo?.description ?? null,
        expiresAt: promo?.validUntil ?? null,
      };
    },
  },
  Query: {
    validatePromoCode: async (_, { code, userId }, { req }) => {
      const authUserId = userId ?? getUserIdFromHeaders(req);
      const [isValid, promo] = authUserId
        ? await Promise.all([validatePromoCodeViaApi(code, authUserId), fetchPromoByCode(code)])
        : [false, await fetchPromoByCode(code)];

      const promoDiscountNumber = promo?.discount == null ? 0 : Number(promo.discount);
      const promoDiscount = Number.isFinite(promoDiscountNumber) ? promoDiscountNumber : 0;

      return {
        isValid,
        originalDiscount: promoDiscount,
        finalDiscount: isValid ? promoDiscount : 0,
        description: promo?.description ?? null,
        expiresAt: promo?.validUntil ?? null,
      };
    },
  },
};

const server = new ApolloServer({
  schema: buildSubgraphSchema([{ typeDefs, resolvers }]),
});

startStandaloneServer(server, {
  listen: { port: 4003 },
  context: async ({ req }) => ({ req }),
}).then(() => {
  console.log('✅ Promo subgraph ready at http://localhost:4003/');
});
