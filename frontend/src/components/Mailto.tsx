type MailtoProps = {
  mailto: string;
  label: string;
};

export default function Mailto({ mailto, label }: MailtoProps) {
  return (
    <a href={`mailto:${mailto}`}
      className="text-muted hover:text-foreground transition-colors duration-150"
      >
      {label}
    </a>
  );
}
