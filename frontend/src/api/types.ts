// shapes returned by the backend API
// use `import type { Listing } from "../api/types";` when needed
export interface Listing {
  id: number;
  title: string;
  description: string;
  category: string;
  price: number;
  quantity: number;
  unit: string;
}

// NOTE: there is no response envelope. endpoints return the payload directly
// and signal success/failure through the HTTP status code. errors come back as
// { error, details } - the interceptor is where this will be normalised
