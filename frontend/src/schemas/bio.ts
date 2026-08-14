import { z } from "zod";
import { bioSchema } from "./common";

export type BioFormValues = z.infer<typeof bioSchema>;
