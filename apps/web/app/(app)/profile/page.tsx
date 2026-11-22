import type { Metadata } from "next";

import { ProfileView } from "@/components/views/profile-view";

export const metadata: Metadata = { title: "Profil" };

export default function ProfilePage() {
  return <ProfileView />;
}
