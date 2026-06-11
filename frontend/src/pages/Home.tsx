import { useEffect, useState } from "react";
import { api } from "../api/client";

export default function Home() {
  const [status, setStatus] = useState("");

  useEffect(() => {
    api.get("/health")
      .then(res => setStatus(res.data))
      .catch(err => setStatus("Backend not reachable"));
  }, []);

  return (
    <div>
      <h1>Marketplace</h1>
      <p>Backend status: {status}</p>
    </div>
  );
}