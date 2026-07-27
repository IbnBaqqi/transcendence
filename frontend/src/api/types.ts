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

// the backend wraps every response like { success, data }.
export interface ApiResponse<T> {
  success: boolean;
  data: T | null;
}
