import { useEffect, useState } from "react";
import { api } from "../api/client";

export default function Home() {
  const [status, setStatus] = useState("");

  useEffect(() => {
    api
      .get("/health")
      .then((res) => setStatus(res.data))
      .catch(() => setStatus("Backend not reachable"));
  }, []);

  return (
    <div className="mx-auto max-w-2xl p-8">
      <h1 className="text-accent text-3xl font-bold">Forage Marketplace</h1>
      <p className="text-muted mt-2">Backend status: {status}</p>
    </div>
  );
}
