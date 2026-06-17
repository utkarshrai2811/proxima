import { GraphQLClient } from 'graphql-request'

// The Go backend serves the GraphQL API at /api/graphql/ on the same origin,
// so same-origin cookies (the proxima_key session cookie) are sent along.
export const gqlClient = new GraphQLClient('/api/graphql/', {
  credentials: 'same-origin',
})
