export const RestoreTypeIcon = ({ type }: { type: 'database' | 'folder' }) => {
  const isDatabase = type === 'database';
  const title = isDatabase ? 'Database' : 'Folder';

  return (
    <div className="flex items-center gap-2 group cursor-help">
      <svg
        viewBox="0 0 24 24"
        className="w-5 h-5"
        fill="currentColor"
      >
        {isDatabase ? (
          <path d="M12 3C7.58 3 4 4.79 4 7s3.58 4 8 4 8-1.79 8-4-3.58-4-8-4zM4 9v3c0 2.21 3.58 4 8 4s8-1.79 8-4V9c0 2.21-3.58 4-8 4s-8-1.79-8-4zm0 5v3c0 2.21 3.58 4 8 4s8-1.79 8-4v-3c0 2.21-3.58 4-8 4s-8-1.79-8-4z" />
        ) : (
          <path d="M10 4H4c-1.1 0-1.99.9-1.99 2L2 18c0 1.1.9 2 2 2h16c1.1 0 2-.9 2-2V8c0-1.1-.9-2-2-2h-8l-2-2z" />
        )}
      </svg>
      <span className="opacity-0 group-hover:opacity-100 absolute bg-base-200 text-sm px-2 py-1 rounded z-10">
        {title}
      </span>
    </div>
  );
};
