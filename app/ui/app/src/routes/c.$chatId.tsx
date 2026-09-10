import { createFileRoute } from "@tanstack/react-router";
import Chat from "@/components/Chat";

export const Route = createFileRoute("/c/$chatId")({
  component: RouteComponent,
});

function RouteComponent() {
  const { chatId } = Route.useParams();

    return (
        <Chat chatId={chatId} />
    );
}
