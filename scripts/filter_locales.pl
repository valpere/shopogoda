#!/usr/bin/env perl

=head1 NAME

filter_locales.pl - Filter locale files to keep only used localization keys

=head1 SYNOPSIS

    perl filter_locales.pl

=head1 DESCRIPTION

This script filters all locale JSON files to keep only the localization keys
that are actually used in the codebase. It reads the list of used keys from
11_keys.csv and removes all unused keys from the locale files.

=cut

use strict;
use warnings;
use utf8;
use JSON::PP;
use File::Spec;
use File::Basename;
use Cwd 'abs_path';

# Enable UTF-8 for all I/O
binmode(STDOUT, ':encoding(UTF-8)');
binmode(STDERR, ':encoding(UTF-8)');

# Find project root
my $script_dir = dirname(abs_path($0));
my $project_root = dirname($script_dir);

# Define directories and files
my $json_dir = File::Spec->catdir($project_root, 'internal', 'locales');
my $used_keys_file = File::Spec->catfile($project_root, 'locales', 'keys-cod.csv');

print "Project root: $project_root\n";
print "JSON directory: $json_dir\n";
print "Used keys file: $used_keys_file\n";

# Read the list of used keys
unless (-f $used_keys_file) {
    die "Used keys file not found: $used_keys_file";
}

my %used_keys;
open my $keys_fh, '<:encoding(utf8)', $used_keys_file
    or die "Cannot open used keys file '$used_keys_file': $!";

while (my $line = <$keys_fh>) {
    chomp $line;
    next if $line eq '' || $line =~ /^\s*$/;  # Skip empty lines
    $used_keys{$line} = 1;
}
close $keys_fh;

my $used_key_count = keys %used_keys;
print "Loaded $used_key_count used keys\n";

# Process each locale file
my @locales = qw(en de es fr uk);

for my $locale (@locales) {
    my $json_file = File::Spec->catfile($json_dir, "$locale.json");

    unless (-f $json_file) {
        print "Warning: JSON file not found: $json_file\n";
        next;
    }

    print "Processing $locale.json...\n";

    # Read JSON file
    open my $json_fh, '<:encoding(utf8)', $json_file
        or die "Cannot open JSON file '$json_file': $!";

    my $json_content = do { local $/; <$json_fh> };
    close $json_fh;

    my $json = JSON::PP->new;
    my $translations = $json->decode($json_content);

    unless (ref $translations eq 'HASH') {
        die "Invalid JSON format in '$json_file'. Expected hash object.";
    }

    # Filter to keep only used keys
    my %filtered_translations;
    my $original_count = keys %$translations;
    my $removed_count = 0;

    for my $key (keys %$translations) {
        if (exists $used_keys{$key}) {
            $filtered_translations{$key} = $translations->{$key};
        } else {
            $removed_count++;
        }
    }

    my $kept_count = keys %filtered_translations;

    # Create backup
    my $backup_file = "$json_file.backup";
    rename($json_file, $backup_file) or die "Cannot create backup: $!";

    # Write filtered JSON file
    my $output_json = JSON::PP->new->pretty->canonical;

    open my $output_fh, '>:encoding(utf8)', $json_file
        or die "Cannot create JSON file '$json_file': $!";

    print $output_fh $output_json->encode(\%filtered_translations);
    close $output_fh;

    print "  -> Original: $original_count keys\n";
    print "  -> Kept: $kept_count keys\n";
    print "  -> Removed: $removed_count keys\n";
    print "  -> Backup saved as: $backup_file\n";
}

print "\nFiltering complete!\n";
print "All locale files have been filtered to contain only used keys.\n";
print "Backup files (.backup) have been created for safety.\n";

=head1 AUTHOR

ShoPogoda Bot Development Team

=cut
