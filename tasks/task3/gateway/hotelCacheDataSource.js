import { RemoteGraphQLDataSource } from '@apollo/gateway';
import { parse, print, visit } from 'graphql';

function normalizeQuery(query) {
  try {
    const document = parse(query);
    const operation = document.definitions.find((d) => d.kind === 'OperationDefinition');

    const variableMap = new Map();
    let index = 0;
    for (const variableDefinition of operation?.variableDefinitions ?? []) {
      const oldName = variableDefinition.variable.name.value;
      variableMap.set(oldName, `v${index}`);
      index += 1;
    }

    const normalized = visit(document, {
      OperationDefinition(node) {
        return node.name ? { ...node, name: null } : node;
      },
      Variable(node) {
        const nextName = variableMap.get(node.name.value);
        if (!nextName) return node;
        return { ...node, name: { ...node.name, value: nextName } };
      },
      VariableDefinition(node) {
        const nextName = variableMap.get(node.variable.name.value);
        if (!nextName) return node;
        return {
          ...node,
          variable: { ...node.variable, name: { ...node.variable.name, value: nextName } },
        };
      },
    });

    return print(normalized);
  } catch {
    return query;
  }
}

export class HotelCachingDataSource extends RemoteGraphQLDataSource {
  async process({ request, context }) {
    const cache = context?.hotelRequestCache;
    if (!(cache instanceof Map)) return super.process({ request, context });
    if (!request?.query || !request?.variables) return super.process({ request, context });

    const queryKey = normalizeQuery(request.query);
    const keyFor = (id) => `${queryKey}::${String(id)}`;

    const idsFromHotelsByIds = Array.isArray(request.variables.ids)
      ? request.variables.ids.map(String)
      : null;

    const representations = Array.isArray(request.variables.representations)
      ? request.variables.representations
      : null;

    const idsFromEntities =
      representations &&
      representations.every((r) => r && r.__typename === 'Hotel' && r.id != null)
        ? representations.map((r) => String(r.id))
        : null;

    const mode = idsFromEntities ? 'entities' : idsFromHotelsByIds ? 'hotelsByIds' : null;
    if (!mode) return super.process({ request, context });

    const ids = mode === 'entities' ? idsFromEntities : idsFromHotelsByIds;

    const loadMany = async () => {
      const missing = [...new Set(ids)].filter((id) => !cache.has(keyFor(id)));
      if (missing.length > 0) {
        const batchPromise = (async () => {
          const nextRequest = { ...request, variables: { ...request.variables } };
          if (mode === 'entities') {
            const seen = new Set();
            nextRequest.variables.representations = [];
            for (const rep of representations) {
              const id = String(rep.id);
              if (!missing.includes(id)) continue;
              if (seen.has(id)) continue;
              seen.add(id);
              nextRequest.variables.representations.push(rep);
            }
          } else {
            nextRequest.variables.ids = missing;
          }

          const response = await super.process({ request: nextRequest, context });
          const items =
            mode === 'entities' ? response?.data?._entities : response?.data?.hotelsByIds;
          const byId = new Map();
          for (let i = 0; i < missing.length; i += 1) byId.set(missing[i], items?.[i] ?? null);
          return byId;
        })();

        for (const id of missing) {
          cache.set(keyFor(id), batchPromise.then((byId) => byId.get(id) ?? null));
        }
      }

      return Promise.all(ids.map((id) => cache.get(keyFor(id))));
    };

    const items = await loadMany();
    return mode === 'entities'
      ? { data: { _entities: items } }
      : { data: { hotelsByIds: items } };
  }
}
