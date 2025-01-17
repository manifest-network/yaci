-- TODO: Add support for
-- - MsgCreateGroup
-- - MsgUpdateGroupAdmin
-- - MsgCreateGroupPolicy
-- - MsgUpdateGroupPolicyAdmin
-- even not if not used in the UI

-- TODO: Cover cases where the proposal execution fails

CREATE OR REPLACE FUNCTION api.parse_msg_create_group_with_policy(msg JSONB, data JSONB, proposal_id TEXT)
RETURNS JSONB
LANGUAGE SQL STABLE
AS $$
  SELECT jsonb_build_object(
      'members', (
          SELECT jsonb_agg(member->'address')
          FROM jsonb_array_elements(msg->'members') AS member
        ),
      'group_policy_address', (
        SELECT trim(both '"' from attr->>'value')
        FROM jsonb_array_elements(data->'txResponse'->'events') AS e
        CROSS JOIN LATERAL jsonb_array_elements(e->'attributes') AS attr
        WHERE e->>'type' = 'cosmos.group.v1.EventCreateGroupPolicy'
          AND attr->>'key' = 'address'
        LIMIT 1
        )
      )
$$;

CREATE OR REPLACE FUNCTION api.parse_msg_submit_proposal(msg JSONB, proposal_id TEXT)
RETURNS JSONB
LANGUAGE SQL STABLE
AS $$
  SELECT jsonb_build_object(
    'title',               msg->>'title',
    'summary',             msg->>'summary',
    'proposal_id', proposal_id,
    'group_policy_address', msg->>'groupPolicyAddress'
  )
$$;

CREATE OR REPLACE FUNCTION api.parse_msg_vote(msg JSONB)
RETURNS JSONB
LANGUAGE SQL STABLE
AS $$
  SELECT jsonb_build_object(
    'proposal_id', msg->>'proposalId',
    'option', msg->>'option'
  )
$$;

CREATE OR REPLACE FUNCTION api.parse_msg_exec(msg JSONB)
RETURNS JSONB
LANGUAGE SQL STABLE
AS $$
  SELECT jsonb_build_object(
    'proposal_id', msg->>'proposalId'
  )
$$;

CREATE OR REPLACE FUNCTION api.parse_msg_update_group_members(msg JSONB, proposal_id TEXT)
RETURNS JSONB
LANGUAGE SQL STABLE
AS $$
  SELECT jsonb_build_object(
    'proposal_id', proposal_id,
    'members', (
      SELECT jsonb_agg(
        jsonb_build_object(
          'address', member->>'address',
          'weight', member->>'weight',
          'metadata', member->>'metadata'
        )
      )
      FROM jsonb_array_elements(msg->'memberUpdates') AS member
    )
  )
$$;

CREATE OR REPLACE FUNCTION api.parse_msg_update_group_metadata(msg JSONB, proposal_id TEXT)
RETURNS JSONB
LANGUAGE SQL STABLE
AS $$
  SELECT jsonb_build_object(
    'proposal_id', proposal_id,
    'metadata', msg->'metadata'
  )
$$;

CREATE OR REPLACE FUNCTION api.parse_msg_update_group_policy_metadata(msg JSONB, proposal_id TEXT)
RETURNS JSONB
LANGUAGE SQL STABLE
AS $$
  SELECT jsonb_build_object(
    'proposal_id', proposal_id,
    'metadata', msg->'metadata'
  )
$$;

CREATE OR REPLACE FUNCTION api.parse_msg_update_group_policy_decision_policy(msg JSONB, proposal_id TEXT)
RETURNS JSONB
LANGUAGE SQL STABLE
AS $$
  SELECT jsonb_build_object(
    'proposal_id', proposal_id,
    'decision_policy', (
    SELECT
      CASE
        WHEN (msg->'decisionPolicy'->>'@type') = '/cosmos.group.v1.ThresholdDecisionPolicy'
        THEN
          jsonb_build_object(
            'threshold', msg->'decisionPolicy'->>'threshold',
            'voting_period', msg->'decisionPolicy'->'windows'->>'votingPeriod',
            'min_execution_period', msg->'decisionPolicy'->'windows'->>'minExecutionPeriod'
          )
        ELSE
          jsonb_build_object(
            'type', msg->'decisionPolicy'->>'@type',
            'error', 'unsupported decision policy type'
          )
      END
    )
  )
$$;

CREATE OR REPLACE FUNCTION api.parse_msg_withdraw_proposal(msg JSONB)
RETURNS JSONB
LANGUAGE SQL STABLE
AS $$
  SELECT jsonb_build_object(
    'proposal_id', msg->>'proposalId'
  )
$$;

CREATE OR REPLACE FUNCTION api.parse_msg_leave_group(msg JSONB)
RETURNS JSONB
LANGUAGE SQL STABLE
AS $$
  SELECT jsonb_build_object(
    'group_id', msg->>'groupId'
  )
$$;

CREATE OR REPLACE FUNCTION api.parse_group_txs(msg JSONB, data JSONB, proposal_id TEXT)
RETURNS JSONB
LANGUAGE SQL STABLE
AS $$
SELECT
  CASE
    WHEN (msg->>'@type') = '/cosmos.group.v1.MsgSubmitProposal'
    THEN
      api.parse_msg_submit_proposal(msg, proposal_id)

    WHEN (msg->>'@type') = '/cosmos.group.v1.MsgVote'
    THEN
      api.parse_msg_vote(msg)

    WHEN (msg->>'@type') = '/cosmos.group.v1.MsgExec'
    THEN
      api.parse_msg_exec(msg)

    WHEN (msg->>'@type') = '/cosmos.group.v1.MsgUpdateGroupMembers'
    THEN
      api.parse_msg_update_group_members(msg, proposal_id)

    WHEN (msg->>'@type') = '/cosmos.group.v1.MsgCreateGroupWithPolicy'
    THEN
      api.parse_msg_create_group_with_policy(msg, data, proposal_id)

    WHEN (msg->>'@type') = '/cosmos.group.v1.MsgUpdateGroupMetadata'
    THEN
      api.parse_msg_update_group_metadata(msg, proposal_id)

    WHEN (msg->>'@type') = '/cosmos.group.v1.MsgUpdateGroupPolicyMetadata'
    THEN
      api.parse_msg_update_group_policy_metadata(msg, proposal_id)

    WHEN (msg->>'@type') = '/cosmos.group.v1.MsgWithdrawProposal'
    THEN
      api.parse_msg_withdraw_proposal(msg)

    WHEN (msg->>'@type') = '/cosmos.group.v1.MsgLeaveGroup'
    THEN
      api.parse_msg_leave_group(msg)

    ELSE
      jsonb_build_object(
        'type', msg->>'@type',
        'error', 'unsupported group message type'
      )
  END
$$;
