#include <unity.h>

#include "button.h"
#include "reaction_game.h"

ReactionCommand command(bool duel = false, bool s2 = true, bool s3 = false) {
  return ReactionCommand{"command-id", "challenge-id", duel, s2, s3, 2000};
}

void test_delay_before_green_and_solo_result() {
  ReactionGame game;
  TEST_ASSERT_TRUE(game.arm(command(), 1000000, false, false));
  game.update(2999999, false, false, false, false);
  TEST_ASSERT_EQUAL_INT(static_cast<int>(ReactionState::ARMED_RED), static_cast<int>(game.state()));
  TEST_ASSERT_TRUE(game.redOn());
  game.update(3000000, false, false, false, false);
  TEST_ASSERT_TRUE(game.greenOn());
  game.update(3243000, true, false, true, false);
  TEST_ASSERT_EQUAL_STRING("S2", game.outcome().button.c_str());
  TEST_ASSERT_EQUAL_INT(243, game.outcome().reactionMs);
  TEST_ASSERT_FALSE(game.outcome().falseStart);
}

void test_false_start_s2_and_s3() {
  ReactionGame s2;
  TEST_ASSERT_TRUE(s2.arm(command(false, true, false), 0, false, false));
  s2.update(1000000, true, false, true, false);
  TEST_ASSERT_TRUE(s2.outcome().falseStart);
  TEST_ASSERT_EQUAL_STRING("S2", s2.outcome().button.c_str());

  ReactionGame s3;
  TEST_ASSERT_TRUE(s3.arm(command(false, false, true), 0, false, false));
  s3.update(1000000, false, true, false, true);
  TEST_ASSERT_TRUE(s3.outcome().falseStart);
  TEST_ASSERT_EQUAL_STRING("S3", s3.outcome().button.c_str());
}

void test_duel_first_press_wins_deterministically() {
  ReactionGame game;
  TEST_ASSERT_TRUE(game.arm(command(true, true, true), 0, false, false));
  game.update(2000000, false, false, false, false);
  game.update(2210000, false, true, false, true);
  TEST_ASSERT_EQUAL_STRING("S3", game.outcome().button.c_str());
  TEST_ASSERT_EQUAL_INT(210, game.outcome().reactionMs);

  ReactionGame simultaneous;
  TEST_ASSERT_TRUE(simultaneous.arm(command(true, true, true), 0, false, false));
  simultaneous.update(2000000, false, false, false, false);
  simultaneous.update(2200000, true, true, true, true);
  TEST_ASSERT_EQUAL_STRING("S2", simultaneous.outcome().button.c_str());
}

void test_timeout_generates_terminal_outcome() {
  ReactionGame game;
  TEST_ASSERT_TRUE(game.arm(command(), 0, false, false));
  game.update(2000000, false, false, false, false);
  game.update(7000000, false, false, false, false);
  TEST_ASSERT_TRUE(game.outcome().timeout);
  TEST_ASSERT_EQUAL_INT(static_cast<int>(ReactionState::REPORT_PENDING), static_cast<int>(game.state()));
}

void test_held_button_is_ignored_until_release() {
  ReactionGame game;
  TEST_ASSERT_TRUE(game.arm(command(), 0, true, false));
  game.update(1000000, true, false, true, false);
  TEST_ASSERT_EQUAL_INT(static_cast<int>(ReactionState::ARMED_RED), static_cast<int>(game.state()));
  game.update(1200000, false, false, false, false);
  game.update(1300000, true, false, true, false);
  TEST_ASSERT_TRUE(game.outcome().falseStart);
}

void test_report_retries_keep_the_same_outcome() {
  ReactionGame game;
  TEST_ASSERT_TRUE(game.arm(command(), 0, false, false));
  game.update(1000, true, false, true, false);
  TEST_ASSERT_TRUE(game.shouldReport(10));
  game.reportAttempt(10, false);
  TEST_ASSERT_FALSE(game.shouldReport(1009));
  TEST_ASSERT_TRUE(game.shouldReport(1010));
  TEST_ASSERT_EQUAL_STRING("command-id", game.outcome().commandId.c_str());
  game.reportAttempt(1010, true);
  TEST_ASSERT_EQUAL_INT(static_cast<int>(ReactionState::IDLE), static_cast<int>(game.state()));
}

void test_debounce_requires_a_stable_transition() {
  DebouncedButton button(true, 30);
  TEST_ASSERT_FALSE(button.update(true, 0));
  TEST_ASSERT_FALSE(button.update(false, 5));
  TEST_ASSERT_FALSE(button.update(true, 10));
  TEST_ASSERT_FALSE(button.update(false, 15));
  TEST_ASSERT_FALSE(button.update(false, 44));
  TEST_ASSERT_TRUE(button.update(false, 45));
  TEST_ASSERT_TRUE(button.pressed());
  TEST_ASSERT_FALSE(button.update(true, 50));
  TEST_ASSERT_FALSE(button.update(true, 80));
  TEST_ASSERT_FALSE(button.pressed());
}

int main(int, char**) {
  UNITY_BEGIN();
  RUN_TEST(test_delay_before_green_and_solo_result);
  RUN_TEST(test_false_start_s2_and_s3);
  RUN_TEST(test_duel_first_press_wins_deterministically);
  RUN_TEST(test_timeout_generates_terminal_outcome);
  RUN_TEST(test_held_button_is_ignored_until_release);
  RUN_TEST(test_report_retries_keep_the_same_outcome);
  RUN_TEST(test_debounce_requires_a_stable_transition);
  return UNITY_END();
}
