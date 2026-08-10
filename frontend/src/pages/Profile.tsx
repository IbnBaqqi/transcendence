// Stub for #24.
//
// Scope reminder (also on the issue): display name + bio are buildable now.
// Avatar upload needs #14, password change needs auth (#32/#33), payout
// account needs the payments module - render those as visible but disabled
// controls so the page reads as finished without pretending to work.
//
// Don't add a form library here: #47 hasn't picked one yet (React Hook Form +
// Zod is the default suggestion), and whoever does #47 will retrofit this.
export default function Profile() {
  return (
    <div className="mx-auto max-w-3xl px-4 py-8">
      <h1 className="text-foreground text-3xl font-bold">Profile & Settings</h1>
      <p className="text-muted mt-2">
        {/* TODO(#24): display name + bio form. The backend doesn't create a
            profiles row on signup yet, so there's nothing to load - hardcode
            values for now. */}
        Under construction (#24).
        <svg><use href="/icons.svg#profile-icon" width="32" height="32"/></svg>
      </p>
    </div>
  );
}
