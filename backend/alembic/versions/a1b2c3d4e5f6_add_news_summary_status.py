"""add news.summary_status

Revision ID: a1b2c3d4e5f6
Revises: dc9ccd427de0
Create Date: 2026-06-06 18:30:00.000000

"""
from typing import Sequence, Union

from alembic import op
import sqlalchemy as sa

# revision identifiers, used by Alembic.
revision: str = 'a1b2c3d4e5f6'
down_revision: Union[str, Sequence[str], None] = 'dc9ccd427de0'
branch_labels: Union[str, Sequence[str], None] = None
depends_on: Union[str, Sequence[str], None] = None


def upgrade() -> None:
    """Upgrade schema."""
    # server_default backfills existing rows to 'pending' as part of the DDL.
    op.add_column(
        'news',
        sa.Column(
            'summary_status',
            sa.String(length=16),
            nullable=False,
            server_default='pending',
        ),
    )
    op.create_check_constraint(
        'ck_news_summary_status',
        'news',
        "summary_status in ('pending','done','failed','skipped')",
    )


def downgrade() -> None:
    """Downgrade schema."""
    op.drop_constraint('ck_news_summary_status', 'news', type_='check')
    op.drop_column('news', 'summary_status')
