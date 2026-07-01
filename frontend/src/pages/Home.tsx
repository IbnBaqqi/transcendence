import { useEffect, useState } from "react";
import { api } from "../api/client";

export default function Home() {
  const [status, setStatus] = useState("");

  useEffect(() => {
    api.get("/health")
      .then(res => setStatus(res.data))
      .catch(() => setStatus("Backend not reachable"));
  }, []);

  return (
    <div className="mx-auto max-w-2xl p-8">
      <h1 className="text-3xl font-bold text-accent">Forage Marketplace</h1>
      <p className="mt-2 text-muted">Backend status: {status}</p>
    </div>
  );
}